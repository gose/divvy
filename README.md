## The Data Game

Let's explore Divvy, Chicago's bike sharing program:

| Trip ID | Bike ID | Start Time | End Time | From Station | To Station | Rider |
|---|---|---|---|---|---|---|
| 123 | 456 | 10/30/2017 13:00:00 | 10/30/2017 13:10:00 | 41.991178, -87.683593 |  41.78101637196, -87.5761197602 | Subscriber / Casual |

### Rules of the Game

1. Ask questions that can be answered by the data.
2. Answer the questions with Kibana.
3. Find surprises in the data.

For example:

* How many Trips were there in 2017?
* What was the most popular weekend to check out a bike?
* What's the most popular bike?
* Did anyone ride across the city?
* Who did it the fastest?

Write down questions you have so we can answer them in Kibana.

### Ingest

Getting this data into Elastic can be accomplished using:

* Logstash
* Beats
* Programming Language

At its core, data is ingested via the [Document APIs](https://www.elastic.co/guide/en/elasticsearch/reference/current/docs.html).  These are a set of RESTful APIs that all of the methods above use to ingest data.  It's recommended you use whatever tool (or language) you are most comfortable with.  Logstash & Beats provide a configuration-driven approach to ingesting data, while a programming langauge will give you more flexibility at the cost of verbosity.  There are tradeoffs to each approach but the choice is yours.

Though Go is not part of the official [Elasticsearch Clients](https://www.elastic.co/guide/en/elasticsearch/client/index.html) supported by Elastic, there is a popular [Elastic Go library](https://github.com/olivere/elastic) that wraps the REST APIs.  We will be using that library to ingest data.

### Data Source

You can download the raw data here:

[https://www.divvybikes.com/system-data](https://www.divvybikes.com/system-data)

Note:

* Trips less than 1 minute in duration are excluded
* Trips greater than 24 hours in duration are excluded
* Some fields (e.g., distance between stations, avg speed) are pre-computed in Go

### Raw Data

There are some variations in the format of the raw data.
In particular, columns change and date formats vary.  You can clean up the data in many different ways before you send it to Elastic for indexing.  You can write a program, use Logstash, Ingest Pipeline, or another ETL tool.  Here we're using some Go code to get it cleaned up and send it into Elastic.

Only the 2018 data sets do not come with a Station list, and some trips in the 2018 data reference new Station IDs.  When we see a trip that does, we exclude it.  But again, this is only for the 2018 data set, not the 2013-2017 data sets.

#### Station Formats

##### Divvy\_Stations\_2013.csv
```
id,name,latitude,longitude,dpcapacity,landmark,online date
5,State St & Harrison St,41.87395806,-87.62773949,19,30,6/28/2013
```

##### Divvy\_Stations\_2014-Q1Q2.csv
```
id,name,latitude,longitude,dpcapacity,online date
43,Michigan Ave & Washington St,41.8838927658,-87.6246491409,43,6/16/13
```

##### Divvy\_Stations\_2014-Q3Q4.csv
```
id,name,latitude,longitude,dpcapacity,dateCreated
5,State St & Harrison St,41.87395806,-87.62773949,19,6/10/2013 10:46
```

##### Divvy\_Stations\_2015.csv
```
id,name,latitude,longitude,dpcapacity,landmark
2,Michigan Ave & Balbo Ave,41.872293,-87.624091,35,541
```

##### Divvy\_Stations\_2016\_Q1Q2.csv
```
id,name,latitude,longitude,dpcapacity,online_date
456,2112 W Peterson Ave,41.991178,-87.683593,15,5/12/2015
```

##### Divvy\_Stations\_2016\_Q3.csv
```
id,name,latitude,longitude,dpcapacity,online_date
456,2112 W Peterson Ave,41.991178,-87.683593,15,5/12/2015
```

##### Divvy\_Stations\_2016\_Q4.csv
```
id,name,latitude,longitude,dpcapacity,online_date
456,2112 W Peterson Ave,41.991178,-87.683593,15,5/12/2015
```

##### Divvy\_Stations\_2017\_Q1Q2.csv
```
"id","name","city","latitude","longitude","dpcapacity","online_date"
"456","2112 W Peterson Ave","Chicago","41.991178","-87.683593","15","2/10/2015 14:04:42"
```

##### Divvy\_Stations\_2017\_Q3Q4.csv
```
id,name,city,latitude,longitude,dpcapacity,online_date,
2,Buckingham Fountain,Chicago,41.876393,-87.620328,27,6/10/2013 10:43,
```


#### Trip Formats

##### Divvy\_Trips\_2013.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthday
4118,2013-06-27 12:11,2013-06-27 12:16,480,316,85,Michigan Ave & Oak St,28,Larrabee St & Menomonee St,Customer,,
```

##### Divvy\_Trips\_2014-Q3-07.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
2886259,7/31/2014 23:56,8/1/2014 0:03,2602,386,291,Wells St & Evergreen Ave,53,Wells St & Erie St,Subscriber,Female,1979
```

##### Divvy\_Trips\_2014-Q3-0809.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
3810750,9/30/2014 23:59,10/1/2014 0:06,851,411,177,Theater on the Lake,143,Sedgwick St & Webster Ave,Subscriber,Male,1982
```

##### Divvy\_Trips\_2014-Q4.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
4413167,12/31/2014 23:54,12/31/2014 23:57,1880,193,296,Broadway & Belmont Ave,334,Lake Shore Dr & Belmont Ave,Subscriber,Male,1989
```

##### Divvy\_Trips\_2014\_Q1Q2.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
2355134,6/30/2014 23:57,7/1/2014 0:07,2006,604,131,Lincoln Ave & Belmont Ave,303,Broadway & Cornelia Ave,Subscriber,Male,1988
```

##### Divvy\_Trips\_2015-Q1.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
4738454,3/31/2015 23:58,4/1/2015 0:03,1095,299,117,Wilton Ave & Belmont Ave,300,Broadway & Barry Ave,Subscriber,Male,1994
```

##### Divvy\_Trips\_2015-Q2.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
5943500,6/30/2015 23:59,7/1/2015 0:10,1712,626,332,Halsted St & Diversey Pkwy,327,Sheffield Ave & Webster Ave,Subscriber,Male,1993
```

##### Divvy\_Trips\_2015\_07.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
6611093,7/31/2015 23:59,8/1/2015 0:03,4829,197,239,Western Ave & Leland Ave,242,Damen Ave & Leland Ave,Subscriber,Male,1978
```

##### Divvy\_Trips\_2015\_08.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
7221407,8/31/2015 23:59,9/1/2015 0:03,4254,245,241,Morgan St & Polk St,21,Aberdeen St & Jackson Blvd,Subscriber,Male,1988
```

##### Divvy\_Trips\_2015\_09.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
7745070,9/30/2015 23:59,10/1/2015 0:04,180,289,229,Southport Ave & Roscoe St,318,Southport Ave & Irving Park Rd,Subscriber,Female,1990
```

##### Divvy\_Trips\_2015\_Q4.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
8547210,12/31/2015 23:48,12/31/2015 23:51,811,166,348,California Ave & 21st St,442,California Ave & 23rd Pl,Subscriber,Male,1978
```

##### Divvy\_Trips\_2016\_04.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
9379901,4/30/2016 23:59,5/1/2016 0:11,21,733,123,California Ave & Milwaukee Ave,374,Western Ave & Walton St,Subscriber,Male,1982
```

##### Divvy\_Trips\_2016\_05.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
9835709,5/31/2016 23:57,6/1/2016 0:14,609,1045,22,May St & Taylor St,282,Halsted St & Maxwell St,Subscriber,Male,1993
```

##### Divvy\_Trips\_2016\_06.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
10426657,6/30/2016 23:59,7/1/2016 0:02,1508,190,93,Sheffield Ave & Willow St,113,Bissell St & Armitage Ave,Subscriber,Male,1993
```

##### Divvy\_Trips\_2016\_Q1.csv
```
trip_id,starttime,stoptime,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
9080551,3/31/2016 23:53,4/1/2016 0:07,155,841,344,Ravenswood Ave & Lawrence Ave,458,Broadway & Thorndale Ave,Subscriber,Male,1986
```

##### Divvy\_Trips\_2016\_Q3.csv
```
"trip_id","starttime","stoptime","bikeid","tripduration","from_station_id","from_station_name","to_station_id","to_station_name","usertype","gender","birthyear"
"12150160","9/30/2016 23:59:58","10/1/2016 00:04:03","4959","245","69","Damen Ave & Pierce Ave","17","Wood St & Division St","Subscriber","Male","1988"
```

##### Divvy\_Trips\_2016\_Q4.csv
```
"trip_id","starttime","stoptime","bikeid","tripduration","from_station_id","from_station_name","to_station_id","to_station_name","usertype","gender","birthyear"
"12979228","12/31/2016 23:57:52","1/1/2017 00:06:44","5076","532","502","California Ave & Altgeld St","258","Logan Blvd & Elston Ave","Customer","",""
```

##### Divvy\_Trips\_2017\_Q1.csv
```
"trip_id","start_time","end_time","bikeid","tripduration","from_station_id","from_station_name","to_station_id","to_station_name","usertype","gender","birthyear"
"13518905","3/31/2017 23:59:07","4/1/2017 00:13:24","5292","857","66","Clinton St & Lake St","171","May St & Cullerton St","Subscriber","Male","1989"
```

##### Divvy\_Trips\_2017\_Q2.csv
```
"trip_id","start_time","end_time","bikeid","tripduration","from_station_id","from_station_name","to_station_id","to_station_name","usertype","gender","birthyear"
"14853213","6/30/2017 23:59:51","7/1/2017 00:13:57","893","846","107","Desplaines St & Jackson Blvd","56","Desplaines St & Kinzie St","Subscriber","Male","1975"
```

##### Divvy\_Trips\_2017\_Q3.csv
```
"trip_id","start_time","end_time","bikeid","tripduration","from_station_id","from_station_name","to_station_id","to_station_name","usertype","gender","birthyear"
"16734065","9/30/2017 23:59:58","10/1/2017 00:05:47","1411","349","216","California Ave & Division St","259","California Ave & Francis Pl","Subscriber","Male","1985"
```

##### Divvy\_Trips\_2017\_Q4.csv
```
trip_id,start_time,end_time,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
17536701,12/31/2017 23:58,1/1/2018 0:03,3304,284,159,Claremont Ave & Hirsch St,69,Damen Ave & Pierce Ave,Subscriber,Male,1988
```

##### Divvy\_Trips\_2018\_0405.csv
```
trip_id,start_time,end_time,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
18709068,5/31/2018 23:59,6/1/2018 0:05,2233,367,21,Aberdeen St & Jackson Blvd,198,Green St & Madison St,Subscriber,Male,1992
```

##### Divvy\_Trips\_2018\_06.csv
```
trip_id,start_time,end_time,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
19244621,2018-06-30 23:59:55,2018-07-01 00:38:10,401,2295,284,Michigan Ave & Jackson Blvd,340,Clark St & Wrightwood Ave,Subscriber,Male,1989
```

##### Divvy\_Trips\_2018\_Q1.csv
```
trip_id,start_time,end_time,bikeid,tripduration,from_station_id,from_station_name,to_station_id,to_station_name,usertype,gender,birthyear
18000526,3/31/2018 23:53,4/1/2018 0:03,2769,570,485,Sawyer Ave & Irving Park Rd,475,Washtenaw Ave & Lawrence Ave,Subscriber,Male,1991
```

