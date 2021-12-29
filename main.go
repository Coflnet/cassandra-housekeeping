package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/gocql/gocql"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog/log"
)

type Trace struct {
	trace_id   string
	span_id    string
	span_hash  string
	duration   string
	start_time string
}

var (
	session  *gocql.Session
	hosts    string
	keyspace string
	username string
	password string
)

func main() {
	readEnvVars()
	hosts := hosts
	cluster := gocql.NewCluster(hosts)
	cluster.Keyspace = keyspace
	cluster.ConnectTimeout = time.Second * 120
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: username,
		Password: password,
	}
	var err error
	session, err = cluster.CreateSession()

	if err != nil {
		log.Error().Err(err).Msgf("something broke in connection")
	}
	defer session.Close()

	before := time.Now().AddDate(0, 0, -2)
	CheckTraces(before)
}

func CheckTraces(before time.Time) {
	row := make(map[string]interface{})

	ctx, _ := context.WithTimeout(context.Background(), time.Duration(time.Minute*1))
	iter := session.Query("SELECT start_time, trace_id, span_id, operation_name FROM jaeger_v1.traces").WithContext(ctx).Iter()
	deleteCount := 0
	var m = make(map[string]int)
	var wg sync.WaitGroup
	for iter.MapScan(row) {
		start_time := row["start_time"].(int64)
		trace_id := row["trace_id"].([]uint8)
		span_id := row["span_id"].(int64)
		operation_name := row["operation_name"].(string)

		if val, ok := m[operation_name]; ok {
			m[operation_name] = val + 1
		} else {
			m[operation_name] = 1
		}

		t := time.UnixMicro(start_time)
		if t.Before(before) {
			wg.Add(1)
			go delete(trace_id, span_id, &wg)
			deleteCount++
		}

		row = make(map[string]interface{})
	}

	wg.Wait()
	log.Info().Msgf("finished deleting %d traces", deleteCount)

	if err := iter.Close(); err != nil {
		log.Fatal().Err(err).Msgf("")
	}
}

func delete(trace_id []byte, span_id int64, wg *sync.WaitGroup) error {
	defer wg.Done()
	err := session.Query("DELETE from jaeger_v1.traces where trace_id = ?", trace_id).Exec()
	if err != nil {
		log.Error().Err(err).Msgf("deleted row %#x, %d", trace_id, span_id)
		return err
	}
	log.Info().Msgf("delete of %#x, %d was successfull", trace_id, span_id)
	return nil
}

func readEnvVars() {
	hosts = os.Getenv("HOSTS")
	username = os.Getenv("USERNAME")
	password = os.Getenv("PASSWORD")
	keyspace = os.Getenv("KEYSPACE")
}
