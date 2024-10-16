package repository

import "github.com/NakarinFIgo/Movies-App/internal/entities"

type DatabaseRepo interface {
	GetUserByEmail(email string) (*entities.User, error)
	GetUserByID(id int) (*entities.User, error)
	AllMovies() ([]*entities.Movie, error)
	AllGenres() ([]*entities.Genre, error)
	InsertMovie(movie entities.Movie) (int, error)
	UpdateMovie(movie entities.Movie) error
	UpdateMovieGenres(id int, genreIDs []int) error
	DeleteMovie(id int) error
	OneMovie(id int) (*entities.Movie, error)
	OneMovieForEdit(id int) (*entities.Movie, []*entities.Genre, error)
}
