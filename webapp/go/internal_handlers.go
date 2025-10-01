package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"time"
)

func matching() {

	query := `
    SELECT chairs.*
    FROM  chairs
    JOIN (
    	SELECT chair_id,latitude,longitude
		FROM (
			SELECT chair_id,latitude,longitude,ROW_NUMBER()
			OVER(PARTITION BY chair_id ORDER BY created_at DESC) AS rn
			FROM chair_locations
		) cl2
		WHERE rn = 1
    ) cl
	ON cl.chair_id = chairs.id
	JOIN chair_models cm ON chairs.model = cm.name
    WHERE
		chairs.is_active = TRUE
		AND NOT EXISTS (
			SELECT 1
  			FROM rides
  		 	JOIN (
    	 		SELECT ride_id
		 	    FROM ride_statuses
    	 		GROUP BY ride_id
    	 		HAVING COUNT(chair_sent_at) < 6
  		 	) AS incomplete_rides
		 	ON incomplete_rides.ride_id = rides.id
  		 	WHERE rides.chair_id = chairs.id
		)
    ORDER BY (ABS(cl.latitude - :pickup_latitude) + ABS(cl.longitude - :pickup_longitude)) ASC
    LIMIT 1
    `
	nstmt, err := db.PrepareNamed(query)
	if err != nil {
		log.Fatal(err)
	}
	defer nstmt.Close()

	for {
		ride := <-rideChannels
		log.Printf("matching start for ride %s\n", ride.ID)

		ctx := context.Background()
		matched := &Chair{}
		if err = nstmt.Get(matched, ride); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("no match found for ride %s\n", ride.ID)
				go func() {
					time.Sleep(time.Second)
					rideChannels <- ride
				}()
				continue
			}
			log.Printf("failed to find match for ride %s: %v\n", ride.ID, err)
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
		sendUserChannel(ride.UserID)
	}
}
