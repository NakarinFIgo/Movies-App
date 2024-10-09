package repository

import "github.com/NakarinFIgo/Movies-App/internal/entities"

type DatabaseRepo interface {
	AllMovies() ([]*entities.Movie, error)
	GetUserByEmail(email string) (*entities.User, error)
	GetUserByID(id int) (*entities.User, error)
	OneMovie(id int) (*entities.Movie, error)
	OneMovieForEdit(id int) (*entities.Movie, []*entities.Genre, error)
}
