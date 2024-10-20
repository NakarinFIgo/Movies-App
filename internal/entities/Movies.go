package entities

import "time"

type Movie struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title"`
	ReleaseDate time.Time `json:"release_date"`
	RunTime     int       `json:"runtime" gorm:"column:runtime"`
	MPAARating  string    `json:"mpaa_rating"`
	Description string    `json:"description"`
	Image       string    `json:"image"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
	Genres      []*Genre  `json:"genres,omitempty" gorm:"many2many:movies_genres"`
	GenresArray []int     `json:"genres_array,omitempty" gorm:"-"`
}

type Genre struct {
	ID    int    `json:"id"`
	Genre string `json:"genre"`
	//Checked   bool      `json:"checked" gorm:"default:false"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
