package main

import (
	"fmt"
	"time"
)

func checkAvailableDate(date time.Time) bool {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	t := time.Date(2020, 1, 1, 0, 0, 0, 0, jst)
	t = t.AddDate(0, 0, availableDays)

	return date.Before(t)
}

func getUsableTrainClassList(fromStation Station, toStation Station) []string {
	usable := map[string]string{}

	for key, value := range TrainClassMap {
		usable[key] = value
	}

	if !fromStation.IsStopExpress {
		delete(usable, "express")
	}
	if !fromStation.IsStopSemiExpress {
		delete(usable, "semi_express")
	}
	if !fromStation.IsStopLocal {
		delete(usable, "local")
	}

	if !toStation.IsStopExpress {
		delete(usable, "express")
	}
	if !toStation.IsStopSemiExpress {
		delete(usable, "semi_express")
	}
	if !toStation.IsStopLocal {
		delete(usable, "local")
	}

	ret := []string{}
	for _, v := range usable {
		ret = append(ret, v)
	}

	return ret
}

/*
index
0: premium not_smoking
1: premium is_smoking
2: reserved not_smoking
3: reserved is_smoking
*/

func (train Train) getAvailableSeats(fromStation Station, toStation Station, seatList []Seat) ([]int, error) {
	// 指定種別の空き座席を返す

	availableSeatMap := map[string]Seat{}
	for _, seat := range seatList {
		availableSeatMap[fmt.Sprintf("%d_%d_%s", seat.CarNumber, seat.SeatRow, seat.SeatColumn)] = seat
	}

	// すでに取られている予約を取得する
	// TODO 遅そう
	query := `
	SELECT sr.reservation_id, sr.car_number, sr.seat_row, sr.seat_column
	FROM seat_reservations sr, reservations r, seat_master s, station_master std, station_master sta
	WHERE
		r.reservation_id=sr.reservation_id AND
		s.train_class=r.train_class AND
		s.car_number=sr.car_number AND
		s.seat_column=sr.seat_column AND
		s.seat_row=sr.seat_row AND
		std.name=r.departure AND
		sta.name=r.arrival
	`

	if train.IsNobori {
		query += "AND ((sta.id < ? AND ? <= std.id) OR (sta.id < ? AND ? <= std.id) OR (? < sta.id AND std.id < ?))"
	} else {
		query += "AND ((std.id <= ? AND ? < sta.id) OR (std.id <= ? AND ? < sta.id) OR (sta.id < ? AND ? < std.id))"
	}

	seatReservationList := []SeatReservation{}
	err := dbx.Select(&seatReservationList, query, fromStation.ID, fromStation.ID, toStation.ID, toStation.ID, fromStation.ID, toStation.ID)
	if err != nil {
		return nil, err
	}

	for _, seatReservation := range seatReservationList {
		key := fmt.Sprintf("%d_%d_%s", seatReservation.CarNumber, seatReservation.SeatRow, seatReservation.SeatColumn)
		delete(availableSeatMap, key)
	}

	ret := []int{0, 0, 0, 0}
	for _, seat := range availableSeatMap {
		if seat.SeatClass == "premium" {
			if !seat.IsSmokingSeat {
				ret[0]++
			} else {
				ret[1]++
			}
		} else if seat.SeatClass == "reserved" {
			if !seat.IsSmokingSeat {
				ret[2]++
			} else {
				ret[3]++
			}
		}
	}

	return ret, nil
}
