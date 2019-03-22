package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olivere/elastic"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Station struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	City       string `json:"city"`
	Latitude   string `json:"latitude"`
	Longitude  string `json:"longitude"`
	TotalDocks int64  `json:"dpcapacity"`
}

type Geo struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

type Trip struct {
	ID                           int64   `json:"id"`
	StartTime                    string  `json:"start_time"`
	EndTime                      string  `json:"end_time"`
	DayOfWeek                    string  `json:"day_of_week"`
	BikeID                       int64   `json:"bike_id"`
	DurationMin                  float64 `json:"duration_min"`
	UserType                     string  `json:"user_type"`
	Gender                       *string `json:"gender"`
	BirthYear                    *int64  `json:"birth_year"`
	Age                          *int64  `json:"age"`
	FromStationID                int64   `json:"from_station_id"`
	FromStationName              string  `json:"from_station_name"`
	FromStationCity              string  `json:"from_station_city"`
	FromStationGeo               Geo     `json:"from_station_geo"`
	FromStationTotalDocks        int64   `json:"from_station_total_docks"`
	ToStationID                  int64   `json:"to_station_id"`
	ToStationName                string  `json:"to_station_name"`
	ToStationCity                string  `json:"to_station_city"`
	ToStationGeo                 Geo     `json:"to_station_geo"`
	ToStationTotalDocks          int64   `json:"to_station_total_docks"`
	DistanceBetweenStationsMiles float64 `json:"distance_between_stations_miles"`
	AverageSpeedMilesPerHour     float64 `json:"average_speed_miles_per_hour"`
}

// checkforErrors is invoked by bulk processor after every commit.
// The err variable indicates success or failure.
func checkForErrors(executionId int64,
	requests []elastic.BulkableRequest,
	response *elastic.BulkResponse,
	err error) {
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Msg("Bulk Insert Error")
		os.Exit(1)
	}
}

// haversin(θ) function
func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	log.Info().Msg("Elastic loader ... starting")

	deleteFlag := flag.Bool("delete", false, "Delete index")
	flag.Parse()

	ctx := context.Background()

	// Elastic Client
	elasticEndpoint := os.Getenv("ELASTIC_ENDPOINT")
	elasticUsername := os.Getenv("ELASTIC_USERNAME")
	elasticPassword := os.Getenv("ELASTIC_PASSWORD")
	log.Info().Str("url", elasticEndpoint).Msg("Elastic server")
	elasticClient, err := elastic.NewClient(elastic.SetURL(elasticEndpoint),
		elastic.SetSniff(false),
		elastic.SetBasicAuth(elasticUsername, elasticPassword))
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("Error initializing Elastic client")
		os.Exit(1)
	}
	log.Info().Msg("Initialized Elastic client")

	indexName := "divvy"

	// Delete
	if *deleteFlag {
		indices := []string{indexName}
		for _, indexName := range indices {
			destroyIndex, err := elasticClient.DeleteIndex(indexName).Do(ctx)
			if err != nil {
				log.Error().
					Str("index", indexName).
					Str("error", err.Error()).
					Msg("Failed to delete Elastic index")
				os.Exit(1)
			} else {
				if !destroyIndex.Acknowledged {
					log.Error().
						Str("index", indexName).
						Str("error", err.Error()).
						Msg("Failed to acknowledge deletion of Elastic index")
					os.Exit(1)
				} else {
					log.Warn().
						Str("index", indexName).
						Msg("Index deleted")
				}
			}
		}
		os.Exit(0)
	}

	// Check for Index, create if necessary
	exists, err := elasticClient.IndexExists(indexName).Do(ctx)
	if err != nil {
		log.Error().
			Str("index", indexName).
			Str("error", err.Error()).
			Msg("Failed to see if index exists")
		os.Exit(1)
	}
	if !exists {
		mapping, err := ioutil.ReadFile("mapping.json")
		createIndex, err := elasticClient.CreateIndex(indexName).
			BodyString(string(mapping)).Do(ctx)
		if err != nil {
			log.Error().
				Str("index", indexName).
				Str("error", err.Error()).
				Msg("Error loading mapping")
			os.Exit(1)
		} else {
			if !createIndex.Acknowledged {
				log.Error().
					Str("index", indexName).
					Str("error", err.Error()).
					Msg("Create index not acknowledged")
				os.Exit(1)
			}
			log.Warn().
				Str("index", indexName).
				Msg("Index created")
		}
	} else {
		log.Warn().
			Str("index", indexName).
			Msg("Index already exists")
	}

	// Bulk Processor
	bulkProc, err := elasticClient.
		BulkProcessor().
		Name("Worker").
		Workers(4).
		After(checkForErrors).
		Do(context.Background())

	// Data Dir
	dataDir := fmt.Sprintf("%s/%s", os.Getenv("HOME"), "data/divvy")

	// Stations
	stations := make(map[int]Station)

	stationFiles := []string{
		"Divvy_Stations_2013.csv",
		"Divvy_Stations_2014-Q1Q2.csv",
		"Divvy_Stations_2014-Q3Q4.csv",
		"Divvy_Stations_2015.csv",
		"Divvy_Stations_2016_Q1Q2.csv",
		"Divvy_Stations_2016_Q3.csv",
		"Divvy_Stations_2016_Q4.csv",
		"Divvy_Stations_2017_Q1Q2.csv",
		"Divvy_Stations_2017_Q3Q4.csv",
	}
	for _, stationFile := range stationFiles {
		log.Debug().
			Str("name", stationFile).
			Msg("Parsing file")
		csvFile, _ := os.Open(fmt.Sprintf("%s/%s", dataDir, stationFile))
		reader := csv.NewReader(bufio.NewReader(csvFile))

		for {
			line, err := reader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Reading Station CSV file")
				os.Exit(1)
			}
			if line[0] == "id" {
				continue
			}
			id, err := strconv.Atoi(line[0])
			if err != nil {
				log.Error().
					Str("id", line[0]).
					Str("error", err.Error()).
					Msg("Converting Station ID to int")
				os.Exit(1)
			}

			var city, latitude, longitude, docks string
			if stationFile == "Divvy_Stations_2017_Q1Q2.csv" ||
				stationFile == "Divvy_Stations_2017_Q3Q4.csv" {
				city = strings.TrimSpace(line[2])
				latitude = line[3]
				longitude = line[4]
				docks = line[5]
			} else {
				city = "Chicago"
				latitude = line[2]
				longitude = line[3]
				docks = line[4]
			}

			// Convert TotalDocks from String to Int64
			totalDocks, err := strconv.Atoi(docks)
			if err != nil {
				log.Error().
					Str("id", line[5]).
					Str("error", err.Error()).
					Msg("Converting TotalDocks to int")
				os.Exit(1)
			}

			stations[id] = Station{
				ID:         int64(id),
				Name:       line[1],
				City:       city,
				Latitude:   latitude,
				Longitude:  longitude,
				TotalDocks: int64(totalDocks),
			}
		}
		log.Debug().
			Int("count", len(stations)).
			Msg("Stations count")
	}
	log.Debug().
		Int("total", len(stations)).
		Msg("Stations total")

	// Trips
	tripFiles := []string{
		//"Divvy_Trips_2018_Q4.csv",
		//"Divvy_Trips_2018_Q3.csv",
		//"Divvy_Trips_2018_Q2.csv",
		//"Divvy_Trips_2018_Q1.csv",
		"Divvy_Trips_2017_Q4.csv",
		"Divvy_Trips_2017_Q3.csv",
		"Divvy_Trips_2017_Q2.csv",
		"Divvy_Trips_2017_Q1.csv",
		//"Divvy_Trips_2016_Q4.csv",
		//"Divvy_Trips_2016_Q3.csv",
		//"Divvy_Trips_2016_06.csv",
		//"Divvy_Trips_2016_05.csv",
		//"Divvy_Trips_2016_04.csv",
		//"Divvy_Trips_2016_Q1.csv",
		//"Divvy_Trips_2015_Q4.csv",
		//"Divvy_Trips_2015_09.csv",
		//"Divvy_Trips_2015_08.csv",
		//"Divvy_Trips_2015_07.csv",
		//"Divvy_Trips_2015-Q2.csv",
		//"Divvy_Trips_2015-Q1.csv",
		//"Divvy_Trips_2014-Q4.csv",
		//"Divvy_Trips_2014-Q3-0809.csv",
		//"Divvy_Trips_2014-Q3-07.csv",
		//"Divvy_Trips_2014_Q1Q2.csv",
		//"Divvy_Trips_2013.csv",
	}

	for _, tripFile := range tripFiles {
		log.Debug().
			Str("name", tripFile).
			Msg("Parsing file")
		csvFile, _ := os.Open(fmt.Sprintf("%s/%s", dataDir, tripFile))
		reader := csv.NewReader(bufio.NewReader(csvFile))

		for {
			line, err := reader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Reading Trip CSV file")
				os.Exit(1)
			}
			if line[0] == "trip_id" {
				continue
			}
			trip := Trip{}

			// Convert Trip ID from String to Int64
			id, err := strconv.Atoi(line[0])
			if err != nil {
				log.Error().
					Str("id", line[0]).
					Str("error", err.Error()).
					Msg("Converting Trip ID to int")
				os.Exit(1)
			}
			trip.ID = int64(id)

			// Convert Trip StartTime from CST to ISO-8601 format
			loc, err := time.LoadLocation("America/Chicago")
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Error getting timezone of America/Chicago")
				os.Exit(1)
			}
			isoTime := fmt.Sprintf("%s %s", line[1], time.Now().In(loc).Format("-0700 MST"))
			parsedTime, err := time.Parse("1/2/2006 15:04:05 -0700 MST", isoTime)
			if err != nil {
				parsedTime, err = time.Parse("1/2/2006 15:04 -0700 MST", isoTime)
				if err != nil {
					parsedTime, err = time.Parse("1/2/2006 -0700 MST", isoTime)
					if err != nil {
						parsedTime, err = time.Parse("2006-01-02 15:04:05 -0700 MST", isoTime)
						if err != nil {
							parsedTime, err = time.Parse("2006-01-02 15:04 -0700 MST", isoTime)
							if err != nil {
								log.Error().
									Str("error", err.Error()).
									Msg("Error parsing StartTime into Time")
								os.Exit(1)
							}
						}
					}
				}
			}
			trip.StartTime = parsedTime.Format(time.RFC3339)

			// Convert Trip EndTime from CST to ISO-8601 format
			loc, err = time.LoadLocation("America/Chicago")
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Error getting timezone of America/Chicago")
				os.Exit(1)
			}
			isoTime = fmt.Sprintf("%s %s", line[2], time.Now().In(loc).Format("-0700 MST"))
			// 3/31/2017 23:59:07
			parsedTime, err = time.Parse("1/2/2006 15:04:05 -0700 MST", isoTime)
			if err != nil {
				parsedTime, err = time.Parse("1/2/2006 15:04 -0700 MST", isoTime)
				if err != nil {
					parsedTime, err = time.Parse("1/2/2006 -0700 MST", isoTime)
					if err != nil {
						parsedTime, err = time.Parse("2006-01-02 15:04:05 -0700 MST", isoTime)
						if err != nil {
							parsedTime, err = time.Parse("2006-01-02 15:04 -0700 MST", isoTime)
							if err != nil {
								log.Error().
									Str("error", err.Error()).
									Msg("Error parsing EndTime into Time")
								os.Exit(1)
							}
						}
					}
				}
			}
			trip.EndTime = parsedTime.Format(time.RFC3339)
			trip.DayOfWeek = parsedTime.Weekday().String()
			//fmt.Printf("%s\n", parsedTime.Format(time.RFC3339))

			// Convert Bike ID from String to Int64
			bikeID, err := strconv.Atoi(line[3])
			if err != nil {
				log.Error().
					Str("id", line[3]).
					Str("error", err.Error()).
					Msg("Converting Bike ID to int")
				os.Exit(1)
			}
			trip.BikeID = int64(bikeID)

			// Convert Duration from String to Float64
			s := strings.Replace(line[4], ",", "", -1)
			dur, err := strconv.ParseFloat(s, 64)
			if err != nil {
				log.Error().
					Str("id", line[4]).
					Str("error", err.Error()).
					Msg("Converting tripduration to float")
				os.Exit(1)
			}
			trip.DurationMin = dur / 60.0

			trip.UserType = line[9]
			if line[9] == "Customer" {
				trip.UserType = "Casual"
			}

			if line[10] != "" {
				gender := line[10]
				trip.Gender = &gender
			}

			// Convert Birth Year from String to Int64
			if string(line[11]) != "" {
				year, err := strconv.Atoi(line[11])
				if err != nil {
					log.Error().
						Int64("id", trip.ID).
						Str("error", err.Error()).
						Msg("Converting Birth Year to int")
					// Instead of exiting, just ignore
				}
				if year > 0 {
					y := int64(year)
					trip.BirthYear = &y
					age := 2018 - y
					trip.Age = &age
				}
			}

			var fromStationLat, fromStationLng, toStationLat, toStationLng string

			// Convert FromStationID from String to Int64
			fromStationID, err := strconv.Atoi(line[5])
			if err != nil {
				log.Error().
					Str("id", line[5]).
					Str("error", err.Error()).
					Msg("Converting FromStationID to int")
				os.Exit(1)
			}
			if station, ok := stations[fromStationID]; ok {
				trip.FromStationID = int64(fromStationID)
				trip.FromStationName = station.Name
				trip.FromStationCity = station.City
				trip.FromStationTotalDocks = station.TotalDocks
				geo := Geo{
					Lat: station.Latitude,
					Lon: station.Longitude,
				}
				trip.FromStationGeo = geo
				fromStationLat = station.Latitude
				fromStationLng = station.Longitude
			} else {
				// Some 2018 data references Station IDs we do not know about.
				continue
			}

			// Convert ToStationID from String to Int64
			toStationID, err := strconv.Atoi(line[7])
			if err != nil {
				log.Error().
					Str("id", line[7]).
					Str("error", err.Error()).
					Msg("Converting ToStationID to int")
				os.Exit(1)
			}
			if station, ok := stations[toStationID]; ok {
				trip.ToStationID = int64(toStationID)
				trip.ToStationName = station.Name
				trip.ToStationCity = station.City
				trip.ToStationTotalDocks = station.TotalDocks
				geo := Geo{
					Lat: station.Latitude,
					Lon: station.Longitude,
				}
				trip.ToStationGeo = geo
				toStationLat = station.Latitude
				toStationLng = station.Longitude
			} else {
				// Some 2018 data references Station IDs we do not know about.
				continue
			}

			fromLat, err := strconv.ParseFloat(fromStationLat, 64)
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Converting From Station Lat to float")
				os.Exit(1)
			}

			fromLng, err := strconv.ParseFloat(fromStationLng, 64)
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Converting From Station Lng to float")
				os.Exit(1)
			}

			toLat, err := strconv.ParseFloat(toStationLat, 64)
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Converting To Station Lat to float")
				os.Exit(1)
			}

			toLng, err := strconv.ParseFloat(toStationLng, 64)
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Converting To Station Lng to float")
				os.Exit(1)
			}

			la1 := fromLat * math.Pi / 180
			lo1 := fromLng * math.Pi / 180
			la2 := toLat * math.Pi / 180
			lo2 := toLng * math.Pi / 180

			earthRadiusMeters := 6378100.0
			h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)
			trip.DistanceBetweenStationsMiles = 2 * earthRadiusMeters *
				math.Asin(math.Sqrt(h)) * 0.000621371

			trip.AverageSpeedMilesPerHour = trip.DistanceBetweenStationsMiles /
				(trip.DurationMin / 60.0)

			// Marshall the trip into JSON and add to queue for Bulk API
			jsonTrip, err := json.Marshal(trip)
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Error marshalling JSON")
				os.Exit(1)
			} else {
				id := fmt.Sprintf("%d", trip.ID)
				indexRequest := elastic.NewBulkIndexRequest().
					Index(indexName).
					Type("_doc").
					OpType("create").
					Id(id).
					Doc(string(jsonTrip))
				bulkProc.Add(indexRequest)
			}
		}
	}

	err = bulkProc.Flush()
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Msg("Flushing Bulk Processor")
		os.Exit(1)
	}
}
