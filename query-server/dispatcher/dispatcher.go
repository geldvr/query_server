package dispatcher

import (
	"bytes"
	"encoding/json"
	"gopkg.in/mgo.v2/bson"
	"io"
	"io/ioutil"
	"itv/query-server/errors"
	"itv/query-server/query"
	"itv/shared/config"
	"itv/shared/db"
	"itv/shared/response"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type dispatcher struct {
	pool    chan chan query.Job
	running bool

	stopDispatcher chan struct{}
	stopWorkers    chan struct{}
	workersStopped chan struct{}
}

var (
	maxQueue   int
	maxWorkers int

	RequestQueue chan query.Job
)

func NewDispatcher() *dispatcher {
	return &dispatcher{
		pool: make(chan chan query.Job, maxWorkers),

		stopDispatcher: make(chan struct{}),

		stopWorkers:    make(chan struct{}),
		workersStopped: make(chan struct{}, maxWorkers),
	}
}

func (d *dispatcher) Run() {
	for i := 0; i < maxWorkers; i++ {
		worker := newWorker(d.pool, d.stopWorkers, d.workersStopped)
		worker.start()
	}

	go d.dispatch()
}

func (d *dispatcher) dispatch() {
	d.running = true
	for {
		select {
		case job := <-RequestQueue:
			go func(job query.Job) {
				workerChannel := <-d.pool
				workerChannel <- job
			}(job)

		case <-d.stopDispatcher:
			log.Print("Stop dispatcher...")
			close(d.stopWorkers)
			finished := 0

			for {
				select {
				case <-d.workersStopped:
					finished++
					if finished == maxWorkers {
						log.Print("Dispatcher's workers are stopped")
						close(d.workersStopped)
						d.running = false
						return
					}
				}

			}
		}
	}
}

func (d *dispatcher) Stop() {
	close(d.stopDispatcher)
	for d.running {
	}

	log.Printf("Dispatcher stopped")
	return
}

type worker struct {
	WorkerPool chan chan query.Job
	JobChannel chan query.Job
	stop       chan struct{}
	notify     chan struct{}
}

func newWorker(pool chan chan query.Job, stop, notify chan struct{}) worker {
	return worker{
		WorkerPool: pool,
		JobChannel: make(chan query.Job),
		stop:       stop,
		notify:     notify,
	}
}

func (w worker) start() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				if err := sendRequest(job); err != nil {
					log.Print(err)
				}

			case <-w.stop:
				w.notify <- struct{}{}
				return
			}
		}
	}()
}

func sendRequest(job query.Job) error {
	var body io.Reader

	defer func() {
		close(job.Wait)
	}()

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	if len(job.Request.Body) != 0 {
		marshaledBody, err := json.Marshal(job.Request.Body)
		if err != nil {
			job.Context.AbortWithStatus(http.StatusInternalServerError)
			return err
		}
		body = bytes.NewBuffer(marshaledBody)
	}

	req, err := http.NewRequest(job.Request.Method, job.Request.Url, body)
	if err != nil {
		job.Context.JSON(http.StatusInternalServerError, errors.RequestCreationError)
		return err
	}

	for key, values := range job.Request.Headers {
		req.Header.Set(key, strings.Join(values, ", "))
	}

	resp, err := client.Do(req)
	if err != nil {
		job.Context.JSON(http.StatusInternalServerError, errors.SendRequestError)
		return err
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	queryResult := query.Query{
		Id:         bson.NewObjectId(),
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Length:     len(bodyBytes),
		CreatedAt:  time.Now(),
	}

	s, d := db.GetMongoInstance()
	defer s.Close()

	if err := d.C("queries").Insert(queryResult); err != nil {
		job.Context.JSON(http.StatusInternalServerError, errors.StoringToDBError)
		return err
	}

	job.Context.JSON(http.StatusOK, response.SuccessResponse(queryResult))
	return nil
}

func init() {
	var err error
	maxQueue, err = strconv.Atoi(config.MaxQueue)
	if err != nil {
		maxQueue = config.GetInt("query-server.maxQueue")
	}
	maxWorkers, err = strconv.Atoi(config.MaxWorker)
	if err != nil {
		maxWorkers = config.GetInt("query-server.maxWorker")
	}

	RequestQueue = make(chan query.Job, maxQueue)
}
