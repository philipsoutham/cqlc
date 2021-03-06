package main

import (
	"fmt"
	"github.com/gocql/gocql"
	"github.com/relops/cqlc/cqlc"
	"github.com/relops/cqlc/integration"
	"log"
	"os"
	"time"
)

func main() {

	session := integration.TestSession("127.0.0.1", "cqlc")
	integration.Truncate(session, EVENTS)

	result := "FAILED"

	ctx := cqlc.NewContext()
	batch := gocql.NewBatch(gocql.LoggedBatch)

	events := 1000

	var first, last gocql.UUID

	for i := 0; i < events; i++ {
		ts := gocql.TimeUUID()
		ctx.Upsert(EVENTS).
			SetInt64(EVENTS.SENSOR, 100).
			SetTimeUUID(EVENTS.TIMESTAMP, ts).
			SetFloat32(EVENTS.TEMPERATURE, 19.8).
			SetInt32(EVENTS.PRESSURE, 357).
			Batch(batch)
		switch i {
		case 0:
			first = ts
		case events - 1:
			last = ts
		}
	}

	if err := session.ExecuteBatch(batch); err != nil {
		log.Fatalf("Could not execute batch: %v", err)
		os.Exit(1)
	}

	count, err := checkOrdering(session, EVENTS.TIMESTAMP)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if count == events {

		count, err = checkOrdering(session, EVENTS.TIMESTAMP.Desc())

		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		if count == events {

			firstRead, err := checkOrderedLimit(session, EVENTS.TIMESTAMP)
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}

			lastRead, err := checkOrderedLimit(session, EVENTS.TIMESTAMP.Desc())
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}

			if first == firstRead {
				if last == lastRead {
					result = "PASSED"
				} else {
					result = fmt.Sprintf("Expected last %v; got %v", last.Time(), lastRead.Time())
				}

			} else {
				result = fmt.Sprintf("Expected first %v; got %v", first.Time(), firstRead.Time())
			}
		}

	} else {
		result = fmt.Sprintf("Expected %d rows; got %d rows", events, count)
	}

	os.Stdout.WriteString(result)
}

func checkOrderedLimit(session *gocql.Session, col cqlc.ClusteredColumn) (gocql.UUID, error) {
	var u gocql.UUID
	ctx := cqlc.NewContext()
	_, err := ctx.Select().
		From(EVENTS).
		Where(EVENTS.SENSOR.Eq(100)).
		OrderBy(col).
		Limit(1).
		Bind(EVENTS.TIMESTAMP.To(&u)).
		FetchOne(session)

	return u, err
}

func checkOrdering(session *gocql.Session, col cqlc.ClusteredColumn) (int, error) {

	ctx := cqlc.NewContext()
	iter, err := ctx.Select().
		From(EVENTS).
		Where(EVENTS.SENSOR.Eq(100)).
		OrderBy(col).
		Fetch(session)

	if err != nil {
		log.Fatalf("Could not execute query: %v", err)
		os.Exit(1)
	}

	count := 0
	var previous time.Time

	err = MapEvents(iter, func(e Events) (bool, error) {

		current := e.Timestamp.Time()

		if !previous.IsZero() {
			if col.IsDescending() {
				if current.After(previous) {
					return false, fmt.Errorf("Wrong ordering (DESC): previous was %v but current is %v", previous, current)
				}
			} else {
				if current.Before(previous) {
					return false, fmt.Errorf("Wrong ordering (ASC): previous was %v but current is %v", previous, current)
				}
			}
		}

		previous = current
		count++
		return true, nil
	})

	return count, err

}
