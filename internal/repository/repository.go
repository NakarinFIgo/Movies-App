package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NakarinFIgo/Movies-App/internal/entities"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	DB *gorm.DB
}

const dbTimeout = time.Second * 5

func (m *PostgresRepository) GetUserByEmail(email string) (*entities.User, error) {

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var user entities.User

	err := m.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (m *PostgresRepository) GetUserByID(id int) (*entities.User, error) {

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var user entities.User

	err := m.DB.WithContext(ctx).Where("id = ?", id).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (m *PostgresRepository) AllMovies() ([]*entities.Movie, error) {

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var movies []*entities.Movie

	result := m.DB.WithContext(ctx).Order("title").Find(&movies)
	if result.Error != nil {
		return nil, result.Error
	}
	return movies, nil
}

func (m *PostgresRepository) InsertUser(user entities.User) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := m.DB.WithContext(ctx).Create(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (m *PostgresRepository) OneMovie(id int) (*entities.Movie, error) {
	var movie entities.Movie

	// Use GORM to find the movie by ID, including preloading genres
	err := m.DB.Preload("Genres").First(&movie, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}

	return &movie, nil
}

func (m *PostgresRepository) OneMovieForEdit(id int) (*entities.Movie, []*entities.Genre, error) {
	var movie entities.Movie

	// ใช้ GORM ในการค้นหาหนังโดย ID
	if err := m.DB.Where("id = ?", id).First(&movie).Error; err != nil {
		return nil, nil, err
	}

	// ดึง genres ที่เกี่ยวข้อง
	var genres []*entities.Genre
	var genresArray []int
	if err := m.DB.Table("movies_genres").
		Select("g.id, g.genre").
		Joins("left join genres g on movies_genres.genre_id = g.id").
		Where("movies_genres.movie_id = ?", id).
		Order("g.genre").
		Scan(&genres).Error; err != nil {
		return nil, nil, err
	}

	for _, g := range genres {
		genresArray = append(genresArray, g.ID)
	}

	movie.Genres = genres
	movie.GenresArray = genresArray

	// ดึง genres ทั้งหมด
	var allGenres []*entities.Genre
	if err := m.DB.Order("genre").Find(&allGenres).Error; err != nil {
		return nil, nil, err
	}

	return &movie, allGenres, nil
}

func (m *PostgresRepository) AllGenres() ([]*entities.Genre, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var genre []*entities.Genre

	result := m.DB.WithContext(ctx).Order("genre").Find(&genre)
	if result.Error != nil {
		return nil, result.Error
	}
	return genre, nil
}

func (m *PostgresRepository) InsertMovie(movie entities.Movie) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := m.DB.WithContext(ctx).Create(&movie).Error; err != nil {
		return 0, err
	}
	return movie.ID, nil
}

func (m *PostgresRepository) UpdateMovie(movie entities.Movie) error {
	if err := m.DB.Model(&movie).Where("id = ?", movie.ID).Updates(entities.Movie{
		Title:       movie.Title,
		Description: movie.Description,
		ReleaseDate: movie.ReleaseDate,
		RunTime:     movie.RunTime,
		MPAARating:  movie.MPAARating,
		UpdatedAt:   movie.UpdatedAt,
		Image:       movie.Image,
	}).Error; err != nil {
		return err
	}
	return nil
}

func (m *PostgresRepository) UpdateMovieGenres(id int, genreIDs []int) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var movie entities.Movie
	if err := m.DB.WithContext(ctx).Begin().First(&movie, id); err != nil {
		m.DB.WithContext(ctx).Begin().Rollback()

	}

	var genres []*entities.Genre
	if err := m.DB.WithContext(ctx).Begin().Where("id IN ?", genreIDs).Find(&genres).Error; err != nil {
		m.DB.WithContext(ctx).Begin().Rollback()
		return err
	}

	if err := m.DB.WithContext(ctx).Begin().Model(&movie).Association("Genres").Replace(genres); err != nil {
		m.DB.WithContext(ctx).Begin().Rollback()
		return err
	}

	return nil
}

func (m *PostgresRepository) DeleteMovie(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var movie entities.Movie
	result := m.DB.WithContext(ctx).First(&movie, id)
	if result.Error != nil {
		return result.Error
	}

	result = m.DB.WithContext(ctx).Delete(&movie)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
