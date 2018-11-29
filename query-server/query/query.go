package query

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Query struct {
	Id         bson.ObjectId       `json:"id" bson:"_id"`
	Status     string              `json:"httpStatus" bson:"status"`
	StatusCode int                 `json:"httpStatus" bson:"statusCode"`
	Headers    map[string][]string `json:"headers" bson:"headers"`
	Length     int                 `json:"length" bson:"length"`
	CreatedAt  time.Time           `json:"-" bson:"createdAt"`
}

type Request struct {
	Method  string                 `json:"method"`
	Url     string                 `json:"url"`
	Headers map[string][]string    `json:"headers"`
	Body    map[string]interface{} `json:"body"`
}

type Job struct {
	Context *gin.Context
	Request *Request
	Wait    chan struct{}
}
