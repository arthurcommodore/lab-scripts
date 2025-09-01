package dto

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Anime struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title            string             `bson:"title" json:"title"`
	Status           string             `bson:"status" json:"status"`
	StartDate        StartDate
	EndDate          EndDate
	Type             string      `bson:"type" json:"type"`
	Episodes         int         `bson:"episodes" json:"episodes"`
	Sources          []string    `bson:"sources" json:"sources"`
	Characters       []Character `bson:"characters" json:"characters"`
	Tags             []string    `bson:"tags" json:"tags"`
	Synopsis         string      `bson:"synopsis" json:"synopsis"`
	Synonyms         []string    `bson:"synonyms" json:"synonyms"`
	Relations        []string    `bson:"relations" json:"relations"`
	PathImage        string      `bson:"pathImage" json:"pathImage"`
	CreatedAt        time.Time   `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time   `bson:"updatedAt" json:"updatedAt"`
	Version          int         `bson:"__v" json:"__v"`
	ChatGpt          bool
	ChatGptDontFound bool
	AverageScore     int
	CountryOfOrigin  string
	IsAdult          bool
	AniListApi       bool
	AniListNotFound  bool
}

type StartDate struct {
	Day   int `bson:"day"`
	Month int `bson:"month"`
	Year  int `bson:"year"`
}

type EndDate struct {
	Day   int `bson:"day"`
	Month int `bson:"month"`
	Year  int `bson:"year"`
}

type AnimeSeason struct {
	Year   int    `bson:"year" json:"year"`
	Season string `bson:"season" json:"season"`
}

type DateOfBirth struct {
	Day   int
	Month int
	Year  int
}

type Character struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	PathImage   string             `bson:"pathImage" json:"pathImage"`
	Link        string             `bson:"link" json:"link"`
	Bio         string             `bson:"bio" json:"bio"`
	Tags        []string           `bson:"tags" json:"tags"`
	Age         string
	DateOfBirth DateOfBirth
	AniListApi  bool
}
