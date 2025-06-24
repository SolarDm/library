package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/project/library/internal/entity"
	"go.uber.org/zap"
)

var _ AuthorRepository = (*postgresImpl)(nil)
var _ BooksRepository = (*postgresImpl)(nil)

type postgresImpl struct {
	logger *zap.Logger
	db     *pgxpool.Pool
}

func NewPostgresRepository(logger *zap.Logger, db *pgxpool.Pool) *postgresImpl {
	return &postgresImpl{
		logger: logger,
		db:     db,
	}
}

func (r *postgresImpl) getRows(bookID string, authorIDs []string) [][]any {
	rows := make([][]any, len(authorIDs))
	for i := range rows {
		rows[i] = []any{&authorIDs[i], &bookID}
	}
	return rows
}

func (r *postgresImpl) mapErr(err error) error {
	const ErrForeignKeyViolation = "23503"

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == ErrForeignKeyViolation {
			return entity.ErrAuthorNotFound
		}
	}

	r.logger.Error("Error while accessing to data base.", zap.Error(err))
	return err
}

func (r *postgresImpl) addBookAuthors(ctx context.Context, tx pgx.Tx, book entity.Book) error {
	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"author_book"},
		[]string{"author_id", "book_id"},
		pgx.CopyFromRows(r.getRows(book.ID, book.AuthorIDs)),
	)

	if err != nil {
		return r.mapErr(err)
	}

	return nil
}

func (r *postgresImpl) getBookFromRows(row pgx.Row) (entity.Book, error) {
	var book entity.Book
	bookAuthors := make([]*string, 0)
	err := row.Scan(&book.ID, &book.Name, &book.CreatedAt, &book.UpdatedAt, &bookAuthors)
	if err != nil {
		r.logger.Error("Error while accessing to data base.", zap.Error(err))
		return entity.Book{}, err
	}

	for _, author := range bookAuthors {
		if author != nil {
			book.AuthorIDs = append(book.AuthorIDs, *author)
		}
	}

	return book, nil
}

func (r *postgresImpl) txRollback(ctx context.Context, tx pgx.Tx) {
	err := tx.Rollback(ctx)
	if err != nil {
		r.logger.Debug("Error while doing rollback.", zap.Error(err))
	}
}

func (r *postgresImpl) AddBook(ctx context.Context, book entity.Book) (resultBook entity.Book, txErr error) {
	var (
		tx  pgx.Tx
		err error
	)

	if tx, err = extractTx(ctx); err != nil {
		tx, err = r.db.Begin(ctx)

		if err != nil {
			return entity.Book{}, err
		}

		defer func() {
			if txErr != nil {
				err = tx.Rollback(ctx)
				r.logger.Error("Error while doing rollback.", zap.Error(err))
				return
			}

			txErr = tx.Commit(ctx)
			if err != nil {
				r.logger.Error("Error while commiting transaction.", zap.Error(err))
			}
		}()
	}

	const queryBook = `INSERT INTO book (name) VALUES ($1) RETURNING id, created_at, updated_at`
	err = tx.QueryRow(ctx, queryBook, book.Name).Scan(&book.ID, &book.CreatedAt, &book.UpdatedAt)

	if err != nil {
		r.logger.Error("Error while accessing to data base.", zap.Error(err))
		return entity.Book{}, err
	}

	err = r.addBookAuthors(ctx, tx, book)
	if err != nil {
		return entity.Book{}, err
	}

	return book, nil
}

func (r *postgresImpl) UpdateBook(ctx context.Context, id string, name string, authorIDs []string) (entity.Book, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return entity.Book{}, err
	}

	defer r.txRollback(ctx, tx)

	book := entity.Book{
		ID:        id,
		Name:      name,
		AuthorIDs: authorIDs,
	}

	const queryUpdateBook = `
UPDATE book
SET name=$2 
WHERE id=$1
RETURNING created_at, updated_at
`

	err = tx.QueryRow(ctx, queryUpdateBook, id, name).Scan(&book.CreatedAt, &book.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.Book{}, entity.ErrBookNotFound
	}
	if err != nil {
		r.logger.Error("Error while accessing to data base.", zap.Error(err))
		return entity.Book{}, err
	}

	const queryDeleteBookAuthors = `DELETE FROM author_book WHERE book_id=ANY($1) AND author_id != ANY($2)`
	_, err = tx.Exec(ctx, queryDeleteBookAuthors, []any{book.ID}, authorIDs)
	if err != nil {
		r.logger.Error("Error while accessing to data base.", zap.Error(err))
		return entity.Book{}, err
	}

	const queryAuthorBooks = `
INSERT INTO author_book
(author_id, book_id)
VALUES ($1, $2)
ON CONFLICT (book_id, author_id) DO NOTHING 
`

	for _, authorID := range book.AuthorIDs {
		_, err = tx.Exec(ctx, queryAuthorBooks, authorID, book.ID)

		if err != nil {
			return entity.Book{}, r.mapErr(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return entity.Book{}, err
	}

	return book, nil
}

func (r *postgresImpl) GetBookInfo(ctx context.Context, id string) (entity.Book, error) {
	const query = `
		SELECT id, name, created_at, updated_at, array_agg(ab.author_id)
		FROM book b
		LEFT JOIN author_book ab on b.id = ab.book_id
		WHERE b.id = $1
		GROUP BY id, name, created_at, updated_at
		`

	book, err := r.getBookFromRows(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return entity.Book{}, entity.ErrBookNotFound
	}
	if err != nil {
		r.logger.Error("Error while accessing to data base.", zap.Error(err))
		return entity.Book{}, err
	}

	return book, nil
}

func (r *postgresImpl) RegisterAuthor(ctx context.Context, author entity.Author) (resultAuthor entity.Author, txErr error) {
	var (
		tx  pgx.Tx
		err error
	)

	if tx, err = extractTx(ctx); err != nil {
		tx, err = r.db.Begin(ctx)

		if err != nil {
			return entity.Author{}, err
		}

		defer func() {
			if txErr != nil {
				err = tx.Rollback(ctx)
				r.logger.Error("Error while doing rollback.", zap.Error(err))
				return
			}

			txErr = tx.Commit(ctx)
			if err != nil {
				r.logger.Error("Error while commiting transaction.", zap.Error(err))
			}
		}()
	}

	const queryAuthor = `INSERT INTO author (name) VALUES ($1) RETURNING id`
	err = tx.QueryRow(ctx, queryAuthor, author.Name).Scan(&author.ID)
	if err != nil {
		r.logger.Error("Error while accessing to data base.", zap.Error(err))
		return entity.Author{}, err
	}

	return author, nil
}

func (r *postgresImpl) ChangeAuthorInfo(ctx context.Context, id string, name string) (entity.Author, error) {
	const queryAuthor = `UPDATE author SET name = $2 WHERE id = $1`
	result, err := r.db.Exec(ctx, queryAuthor, id, name)
	if err != nil {
		r.logger.Error("Error while accessing to data base.", zap.Error(err))
		return entity.Author{}, err
	}
	if result.RowsAffected() == 0 {
		return entity.Author{}, entity.ErrAuthorNotFound
	}

	author := entity.Author{
		ID:   id,
		Name: name,
	}

	return author, nil
}

func (r *postgresImpl) GetAuthorInfo(ctx context.Context, id string) (entity.Author, error) {
	const queryAuthor = `SELECT id, name FROM author WHERE id = ANY($1)`
	var author entity.Author
	err := r.db.QueryRow(ctx, queryAuthor, []any{id}).Scan(&author.ID, &author.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.Author{}, entity.ErrAuthorNotFound
	}
	if err != nil {
		r.logger.Error("Error while accessing to data base.", zap.Error(err))
		return entity.Author{}, err
	}

	return author, nil
}

func (r *postgresImpl) GetAuthorBooks(ctx context.Context, id string) ([]entity.Book, error) {
	const query = `
		SELECT id, name, created_at, updated_at, array_agg(ab.author_id)
		FROM book b
		LEFT JOIN author_book ab on b.id = ab.book_id
		WHERE b.id = ANY (
		    SELECT ids.book_id 
		    FROM author_book ids
		    WHERE ids.author_id = $1
		)
		GROUP BY id, name, created_at, updated_at
		`

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		r.logger.Error("Error while accessing to data base.", zap.Error(err))
		return nil, err
	}

	defer rows.Close()

	authorBooks := make([]entity.Book, 0)

	for rows.Next() {
		book, err := r.getBookFromRows(rows)
		if err != nil {
			r.logger.Error("Error while working with row.", zap.Error(err))
			return nil, err
		}
		authorBooks = append(authorBooks, book)
	}
	return authorBooks, nil
}
