package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…
	ride := &Ride{}
	if err := db.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 1`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	matched := &Chair{}
	if err := db.GetContext(ctx, matched, `
	    SELECT chairs.*
	    FROM chairs
	    JOIN (
	      SELECT cl1.chair_id, cl1.latitude, cl1.longitude
	      FROM chair_locations cl1
	      WHERE cl1.created_at = (
	        SELECT MAX(cl2.created_at) FROM chair_locations cl2 WHERE cl2.chair_id = cl1.chair_id
	      )
	    ) cl ON cl.chair_id = chairs.id
	    WHERE chairs.is_active = TRUE
	      AND NOT EXISTS (
	        SELECT 1 FROM rides
	        WHERE chair_id = chairs.id
	          AND EXISTS (
	            SELECT 1 FROM (
	              SELECT ride_id, COUNT(chair_sent_at) AS cnt
	              FROM ride_statuses
	              WHERE ride_id = rides.id
	              GROUP BY ride_id
	            ) AS t
	            WHERE t.ride_id = rides.id AND t.cnt < 6
	          )
	      )
	    ORDER BY (ABS(cl.latitude - ?) + ABS(cl.longitude - ?)) ASC
	    LIMIT 1
	    `, ride.PickupLatitude, ride.PickupLongitude); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("no match found for ride %s\n", ride.ID)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	log.Printf("matched ride %s with chair %s\n", ride.ID, matched.ID)

	if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matched.ID, ride.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	channel, ok := chairChannels.getChannel(matched.ID)
	if ok {
		fmt.Printf("notify send chair channel found: %s\n", matched.ID)
		channel <- struct{}{}
	} else {
		fmt.Printf("notify send chair channel not found: %s\n", matched.ID)
	}

	w.WriteHeader(http.StatusNoContent)
}

func matching() {

	for {
		ride := <-rideChannels
		log.Printf("matching start for ride %s\n", ride.ID)

		ctx := context.Background()
		matched := &Chair{}
		if err := db.GetContext(ctx, matched, `
    SELECT chairs.*
    FROM chairs
    JOIN (
      SELECT cl1.chair_id, cl1.latitude, cl1.longitude
      FROM chair_locations cl1
      WHERE cl1.created_at = (
        SELECT MAX(cl2.created_at) FROM chair_locations cl2 WHERE cl2.chair_id = cl1.chair_id
      )
    ) cl ON cl.chair_id = chairs.id
    WHERE chairs.is_active = TRUE
      AND NOT EXISTS (
        SELECT 1 FROM rides
        WHERE chair_id = chairs.id
          AND EXISTS (
            SELECT 1 FROM (
              SELECT ride_id, COUNT(chair_sent_at) AS cnt
              FROM ride_statuses
              WHERE ride_id = rides.id
              GROUP BY ride_id
            ) AS t
            WHERE t.ride_id = rides.id AND t.cnt < 6
          )
      )
    ORDER BY (ABS(cl.latitude - ?) + ABS(cl.longitude - ?)) ASC
    LIMIT 1
    `, ride.PickupLatitude, ride.PickupLongitude); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("no match found for ride %s\n", ride.ID)
				go func() {
					time.Sleep(time.Second)
					rideChannels <- ride
				}()
				continue
			}
			continue
		}

		if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matched.ID, ride.ID); err != nil {
			log.Printf("failed to update ride %s: %v\n", ride.ID, err)
			continue
		}
		channel, ok := chairChannels.getChannel(matched.ID)
		if ok {
			fmt.Printf("notify send chair channel found: %s\n", matched.ID)
			channel <- struct{}{}
		} else {
			fmt.Printf("notify send chair channel not found: %s\n", matched.ID)
		}
	}
}
