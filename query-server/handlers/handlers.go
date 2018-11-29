package handlers

import (
	"bytes"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"itv/query-server/dispatcher"
	"itv/query-server/errors"
	"itv/query-server/query"
	"itv/shared/config"
	"itv/shared/db"
	"itv/shared/response"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	QueryTimeFormat = "2006-01-02T15:04:05"
	queryPattern    = "page={{.page}}&limit={{.limit}}{{if .from}}&from={{.from}}{{end}}{{if .to}}&to={{.to}}{{end}}"
)

var (
	queryTemplate = template.Must(template.New("").Parse(queryPattern))
)

type paginated struct {
	Base    string        `json:"base"`
	Prev    string        `json:"prev,omitempty"`
	Current string        `json:"current,omitempty"`
	Next    string        `json:"next,omitempty"`
	Limit   int           `json:"limit"`
	Size    int           `json:"size"`
	Results []query.Query `json:"results"`
}

type filter struct {
	RegisteredAt string `form:"from"`
	RegisteredTo string `form:"to"`
	Page         string `form:"page"`
	Limit        string `form:"limit"`
	page         int
	limit        int
}

const (
	ALL     = 0
	FROM    = 1
	TO      = 2
	FROM_TO = 3
)

func (f *filter) getPipe() ([]bson.M, []*response.Error) {
	var (
		mode int
		from time.Time
		to   time.Time

		err  error
		errs = make([]*response.Error, 0, 3)
		pipe = make([]bson.M, 0, 3)
	)

	if f.page, err = strconv.Atoi(f.Page); err != nil && f.Page != "" {
		errs = append(errs, &response.Error{Field:"page", Message: "must be int"})
	}

	if f.limit, err = strconv.Atoi(f.Limit); err != nil && f.Limit != "" {
		errs = append(errs, &response.Error{Field:"limit", Message: "must be int"})
	}

	if f.limit <= 0 {
		f.limit = 100
	}

	if f.page <= 0 {
		f.page = 1
	}

	skip := bson.M{
		"$skip": (f.page - 1) * f.limit,
	}

	limit := bson.M{
		"$limit": f.limit + 1,
	}

	sort := bson.M{
		"$sort": bson.M{
			"createdAt": 1,
		},
	}

	matchReg := bson.M{}
	if len(f.RegisteredAt) > 0 {
		f.RegisteredAt = strings.TrimSpace(f.RegisteredAt)

		if from, err = time.ParseInLocation(QueryTimeFormat, f.RegisteredAt, time.Local); err != nil {
			errs = append(errs, &response.Error{Field: "from", Message: fmt.Sprintf("invalid date format[%s]", QueryTimeFormat)})
		} else {
			matchReg["$gte"] = from
			mode |= FROM
		}
	}

	if len(f.RegisteredTo) > 0 {
		f.RegisteredTo = strings.TrimSpace(f.RegisteredTo)

		if to, err = time.ParseInLocation(QueryTimeFormat, f.RegisteredTo, time.Local); err != nil {
			errs = append(errs, &response.Error{Field: "to", Message: fmt.Sprintf("invalid date format[%s]", QueryTimeFormat)})
		} else {
			matchReg["$lte"] = to
			mode |= TO
		}
	}

	if mode == FROM_TO && from.After(to) {
		errs = append(errs, &response.Error{Field: "from, to", Message: "invalid values[to < from]"})
	}

	if len(errs) != 0 {
		return nil, errs
	}

	if len(matchReg) != 0 {
		match := bson.M{}
		match["createdAt"] = matchReg
		pipe = append(pipe, bson.M{"$match": match})
	}

	return append(pipe, []bson.M{sort, skip, limit}...), nil
}

func (f filter) nextPage() string {
	f.page += 1
	return f.getQuery()
}

func (f filter) prevPage() string {
	f.page -= 1
	return f.getQuery()
}

func (f filter) curPage() string {
	return f.getQuery()
}

func (f filter) getQuery() string {
	buf := &bytes.Buffer{}
	m := map[string]interface{}{
		"from":  f.RegisteredAt,
		"to":    f.RegisteredTo,
		"limit": f.limit,
		"page":  f.page,
	}

	queryTemplate.Execute(buf, m)
	return "/api/queries?" + buf.String()
}

func CreateQuery(c *gin.Context) {
	request := new(query.Request)
	if err := c.BindJSON(request); err != nil {
		c.JSON(http.StatusBadRequest, errors.BindingJSONError)
		return
	}

	errs := make([]*response.Error, 0, 2)
	if !govalidator.IsURL(request.Url) {
		errs = append(errs, &response.Error{Field: "url", Message: "invalid value"})
	}

	request.Method = strings.ToUpper(strings.TrimSpace(request.Method))
	switch request.Method {
	case "":
		request.Method = http.MethodGet
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
	default:
		errs = append(errs, &response.Error{Field: "method", Message: "invalid value"})
	}

	if len(errs) != 0 {
		c.JSON(http.StatusBadRequest, response.ErrorResponse(errs...))
		return
	}

	wait := make(chan struct{})
	dispatcher.RequestQueue <- query.Job{c, request, wait}
	<-wait
}

func GetQueryById(c *gin.Context) {
	id := c.Param("id")

	if bson.IsObjectIdHex(id) == false {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	s, d := db.GetMongoInstance()
	defer s.Close()

	query := new(query.Query)
	if err := d.C("queries").Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(query); err != nil {
		status := http.StatusInternalServerError
		if err == mgo.ErrNotFound {
			status = http.StatusNotFound
		}

		c.AbortWithStatus(status)
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(query))
}

func GetQueries(c *gin.Context) {
	filter := new(filter)
	if err := c.Bind(filter); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse(&response.Error{Message: err.Error()}))
		return
	}

	pipe, errs := filter.getPipe()
	if errs != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse(errs...))
		return
	}

	s, d := db.GetMongoInstance()
	defer s.Close()

	var queries []query.Query
	if err := d.C("queries").Pipe(pipe).All(&queries); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	pd := paginated{
		Base:  config.BaseUrl,
		Limit: filter.limit,
	}

	pd.Current = filter.curPage()

	if len(queries) > filter.limit {
		queries = queries[:len(queries)-1]
		pd.Next = filter.nextPage()
	}

	if filter.page > 1 {
		pd.Prev = filter.prevPage()
	}

	pd.Size = len(queries)
	pd.Results = queries

	c.JSON(http.StatusOK, response.SuccessResponse(pd))
}

func DeleteQueryById(c *gin.Context) {
	id := c.Param("id")
	if !bson.IsObjectIdHex(id) {
		c.JSON(http.StatusNotFound, errors.QueryNotFoundInDB)
		return
	}
	s, d := db.GetMongoInstance()
	defer s.Close()

	if err := d.C("queries").Remove(bson.M{
		"_id": bson.ObjectIdHex(id),
	}); err != nil {
		if err == mgo.ErrNotFound {
			c.JSON(http.StatusNotFound, errors.QueryNotFoundInDB)
			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(bson.M{
		"id": id,
	}))
}
