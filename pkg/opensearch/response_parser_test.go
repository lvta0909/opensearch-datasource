package opensearch

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/opensearch-datasource/pkg/opensearch/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ResponseParser_test(t *testing.T) {
	t.Run("Simple query and count", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
					"timeField": "@timestamp",
					"metrics": [{ "type": "count", "id": "1" }],
          "bucketAggs": [{ "type": "date_histogram", "field": "@timestamp", "id": "2" }]
				}`,
		}
		response := `{
        "responses": [
          {
            "aggregations": {
              "2": {
                "buckets": [
                  {
                    "doc_count": 10,
                    "key": 1000
                  },
                  {
                    "doc_count": 15,
                    "key": 2000
                  }
                ]
              }
            }
          }
        ]
			}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		assert.Len(t, queryRes.Frames, 1)
		series := queryRes.Frames[0]
		assert.Equal(t, "Count", series.Name)

		require.Len(t, series.Fields, 2)
		require.Equal(t, 2, series.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *series.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *series.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, series.Fields[1].Len())
		assert.EqualValues(t, 10, *series.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 15, *series.Fields[1].At(1).(*float64))
	})

	t.Run("Simple query count & avg aggregation", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
					"timeField": "@timestamp",
					"metrics": [{ "type": "count", "id": "1" }, {"type": "avg", "field": "value", "id": "2" }],
		 "bucketAggs": [{ "type": "date_histogram", "field": "@timestamp", "id": "3" }]
				}`,
		}
		response := `{
	       "responses": [
	         {
	           "aggregations": {
	             "3": {
	               "buckets": [
	                 {
	                   "2": { "value": 88 },
	                   "doc_count": 10,
	                   "key": 1000
	                 },
	                 {
	                   "2": { "value": 99 },
	                   "doc_count": 15,
	                   "key": 2000
	                 }
	               ]
	             }
	           }
	         }
	       ]
				}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		assert.Len(t, queryRes.Frames, 2)

		seriesOne := queryRes.Frames[0]
		assert.Equal(t, "Count", seriesOne.Name)
		require.Len(t, seriesOne.Fields, 2)
		require.Equal(t, 2, seriesOne.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesOne.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesOne.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesOne.Fields[1].Len())
		assert.EqualValues(t, 10, *seriesOne.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 15, *seriesOne.Fields[1].At(1).(*float64))

		seriesTwo := queryRes.Frames[1]
		assert.Equal(t, "Average value", seriesTwo.Name)
		require.Len(t, seriesTwo.Fields, 2)
		require.Equal(t, 2, seriesTwo.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesTwo.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesTwo.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesTwo.Fields[1].Len())
		assert.EqualValues(t, 88, *seriesTwo.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 99, *seriesTwo.Fields[1].At(1).(*float64))
	})

	t.Run("Single group by query one metric", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
					"timeField": "@timestamp",
					"metrics": [{ "type": "count", "id": "1" }],
		 "bucketAggs": [
						{ "type": "terms", "field": "host", "id": "2" },
						{ "type": "date_histogram", "field": "@timestamp", "id": "3" }
					]
				}`,
		}
		response := `{
		   "responses": [
			 {
			   "aggregations": {
				 "2": {
				   "buckets": [
					 {
					   "3": {
						 "buckets": [{ "doc_count": 1, "key": 1000 }, { "doc_count": 3, "key": 2000 }]
					   },
					   "doc_count": 4,
					   "key": "server1"
					 },
					 {
					   "3": {
						 "buckets": [{ "doc_count": 2, "key": 1000 }, { "doc_count": 8, "key": 2000 }]
					   },
					   "doc_count": 10,
					   "key": "server2"
					 }
				   ]
				 }
			   }
			 }
		   ]
				}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		assert.Len(t, queryRes.Frames, 2)
		seriesOne := queryRes.Frames[0]
		assert.Equal(t, "server1", seriesOne.Name)
		require.Len(t, seriesOne.Fields, 2)
		require.Equal(t, 2, seriesOne.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesOne.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesOne.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesOne.Fields[1].Len())
		assert.EqualValues(t, 1, *seriesOne.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 3, *seriesOne.Fields[1].At(1).(*float64))

		seriesTwo := queryRes.Frames[1]
		assert.Equal(t, "server2", seriesTwo.Name)
		require.Len(t, seriesTwo.Fields, 2)
		require.Equal(t, 2, seriesTwo.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesTwo.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesTwo.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesTwo.Fields[1].Len())
		assert.EqualValues(t, 2, *seriesTwo.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 8, *seriesTwo.Fields[1].At(1).(*float64))
	})

	t.Run("Single group by query two metrics", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
				"timeField": "@timestamp",
				"metrics": [{ "type": "count", "id": "1" }, { "type": "avg", "field": "@value", "id": "4" }],
	 "bucketAggs": [
					{ "type": "terms", "field": "host", "id": "2" },
					{ "type": "date_histogram", "field": "@timestamp", "id": "3" }
				]
			}`,
		}
		response := `{
		   "responses": [
			 {
			   "aggregations": {
				 "2": {
				   "buckets": [
					 {
					   "3": {
						 "buckets": [
						   { "4": { "value": 10 }, "doc_count": 1, "key": 1000 },
						   { "4": { "value": 12 }, "doc_count": 3, "key": 2000 }
						 ]
					   },
					   "doc_count": 4,
					   "key": "server1"
					 },
					 {
					   "3": {
						 "buckets": [
						   { "4": { "value": 20 }, "doc_count": 1, "key": 1000 },
						   { "4": { "value": 32 }, "doc_count": 3, "key": 2000 }
						 ]
					   },
					   "doc_count": 10,
					   "key": "server2"
					 }
				   ]
				 }
			   }
			 }
		   ]
				}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		assert.Len(t, queryRes.Frames, 4)
		seriesOne := queryRes.Frames[0]
		assert.Equal(t, "server1 Count", seriesOne.Name)
		require.Len(t, seriesOne.Fields, 2)
		require.Equal(t, 2, seriesOne.Fields[0].Len())
		assert.EqualValues(t, 1, *seriesOne.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 3, *seriesOne.Fields[1].At(1).(*float64))
		require.Equal(t, 2, seriesOne.Fields[1].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesOne.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesOne.Fields[0].At(1).(*time.Time))

		seriesTwo := queryRes.Frames[1]
		assert.Equal(t, "server1 Average @value", seriesTwo.Name)
		require.Len(t, seriesTwo.Fields, 2)
		require.Equal(t, 2, seriesTwo.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesTwo.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesTwo.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesTwo.Fields[1].Len())
		assert.EqualValues(t, 10, *seriesTwo.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 12, *seriesTwo.Fields[1].At(1).(*float64))

		seriesThree := queryRes.Frames[2]
		assert.Equal(t, "server2 Count", seriesThree.Name)
		require.Len(t, seriesThree.Fields, 2)
		require.Equal(t, 2, seriesThree.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesThree.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesThree.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesThree.Fields[1].Len())
		assert.EqualValues(t, 1, *seriesThree.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 3, *seriesThree.Fields[1].At(1).(*float64))

		seriesFour := queryRes.Frames[3]
		assert.Equal(t, "server2 Average @value", seriesFour.Name)
		require.Len(t, seriesFour.Fields, 2)
		require.Equal(t, 2, seriesFour.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesFour.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesFour.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesFour.Fields[1].Len())
		assert.EqualValues(t, 20, *seriesFour.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 32, *seriesFour.Fields[1].At(1).(*float64))
	})

	t.Run("With percentiles", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
					"timeField": "@timestamp",
					"metrics": [{ "type": "percentiles", "settings": { "percents": [75, 90] }, "id": "1" }],
		 "bucketAggs": [{ "type": "date_histogram", "field": "@timestamp", "id": "3" }]
				}`,
		}
		response := `{
		   "responses": [
			 {
			   "aggregations": {
				 "3": {
				   "buckets": [
					 {
					   "1": { "values": { "75": 3.3, "90": 5.5 } },
					   "doc_count": 10,
					   "key": 1000
					 },
					 {
					   "1": { "values": { "75": 2.3, "90": 4.5 } },
					   "doc_count": 15,
					   "key": 2000
					 }
				   ]
				 }
			   }
			 }
		   ]
				}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		assert.Len(t, queryRes.Frames, 2)
		seriesOne := queryRes.Frames[0]
		assert.Equal(t, "p75", seriesOne.Name)
		require.Len(t, seriesOne.Fields, 2)
		require.Equal(t, 2, seriesOne.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesOne.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesOne.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesOne.Fields[1].Len())
		assert.EqualValues(t, 3.3, *seriesOne.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 2.3, *seriesOne.Fields[1].At(1).(*float64))

		seriesTwo := queryRes.Frames[1]
		assert.Equal(t, "p90", seriesTwo.Name)
		require.Len(t, seriesTwo.Fields, 2)
		require.Equal(t, 2, seriesTwo.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesTwo.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesTwo.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesTwo.Fields[1].Len())
		assert.EqualValues(t, 5.5, *seriesTwo.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 4.5, *seriesTwo.Fields[1].At(1).(*float64))
	})

	t.Run("With extended stats", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
					"timeField": "@timestamp",
					"metrics": [{ "type": "extended_stats", "meta": { "max": true, "std_deviation_bounds_upper": true, "std_deviation_bounds_lower": true }, "id": "1" }],
		 "bucketAggs": [
						{ "type": "terms", "field": "host", "id": "3" },
						{ "type": "date_histogram", "field": "@timestamp", "id": "4" }
					]
				}`,
		}
		response := `{
		   "responses": [
			 {
			   "aggregations": {
				 "3": {
				   "buckets": [
					 {
					   "key": "server1",
					   "4": {
						 "buckets": [
						   {
							 "1": {
							   "max": 10.2,
							   "min": 5.5,
							   "std_deviation_bounds": { "upper": 3, "lower": -2 }
							 },
							 "doc_count": 10,
							 "key": 1000
						   }
						 ]
					   }
					 },
					 {
					   "key": "server2",
					   "4": {
						 "buckets": [
						   {
							 "1": {
							   "max": 15.5,
							   "min": 3.4,
							   "std_deviation_bounds": { "upper": 4, "lower": -1 }
							 },
							 "doc_count": 10,
							 "key": 1000
						   }
						 ]
					   }
					 }
				   ]
				 }
			   }
			 }
		   ]
				}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		require.Len(t, queryRes.Frames, 6)

		seriesOne := queryRes.Frames[0]
		assert.Equal(t, "server1 Max", seriesOne.Name)
		require.Len(t, seriesOne.Fields, 2)
		require.Equal(t, 1, seriesOne.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesOne.Fields[0].At(0).(*time.Time))
		require.Equal(t, 1, seriesOne.Fields[1].Len())
		assert.EqualValues(t, 10.2, *seriesOne.Fields[1].At(0).(*float64))

		seriesTwo := queryRes.Frames[1]
		assert.Equal(t, "server1 Std Dev Lower", seriesTwo.Name)
		require.Len(t, seriesTwo.Fields, 2)
		require.Equal(t, 1, seriesTwo.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesTwo.Fields[0].At(0).(*time.Time))
		require.Equal(t, 1, seriesTwo.Fields[1].Len())
		assert.EqualValues(t, -2, *seriesTwo.Fields[1].At(0).(*float64))

		seriesThree := queryRes.Frames[2]
		assert.Equal(t, "server1 Std Dev Upper", seriesThree.Name)
		require.Len(t, seriesThree.Fields, 2)
		require.Equal(t, 1, seriesThree.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesThree.Fields[0].At(0).(*time.Time))
		require.Equal(t, 1, seriesThree.Fields[1].Len())
		assert.EqualValues(t, 3, *seriesThree.Fields[1].At(0).(*float64))

		seriesFour := queryRes.Frames[3]
		assert.Equal(t, "server2 Max", seriesFour.Name)
		require.Len(t, seriesFour.Fields, 2)
		require.Equal(t, 1, seriesFour.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesFour.Fields[0].At(0).(*time.Time))
		require.Equal(t, 1, seriesFour.Fields[1].Len())
		assert.EqualValues(t, 15.5, *seriesFour.Fields[1].At(0).(*float64))

		seriesFive := queryRes.Frames[4]
		assert.Equal(t, "server2 Std Dev Lower", seriesFive.Name)
		require.Len(t, seriesFive.Fields, 2)
		require.Equal(t, 1, seriesFive.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesFive.Fields[0].At(0).(*time.Time))
		require.Equal(t, 1, seriesFive.Fields[1].Len())
		assert.EqualValues(t, -1, *seriesFive.Fields[1].At(0).(*float64))

		seriesSix := queryRes.Frames[5]
		assert.Equal(t, "server2 Std Dev Upper", seriesSix.Name)
		require.Len(t, seriesSix.Fields, 2)
		require.Equal(t, 1, seriesSix.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesSix.Fields[0].At(0).(*time.Time))
		require.Equal(t, 1, seriesSix.Fields[1].Len())
		assert.EqualValues(t, 4, *seriesSix.Fields[1].At(0).(*float64))
	})

	t.Run("Single group by with alias pattern", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
					"timeField": "@timestamp",
					"alias": "{{term @host}} {{metric}} and {{not_exist}} {{@host}}",
					"metrics": [{ "type": "count", "id": "1" }],
		 "bucketAggs": [
						{ "type": "terms", "field": "@host", "id": "2" },
						{ "type": "date_histogram", "field": "@timestamp", "id": "3" }
					]
				}`,
		}
		response := `{
		   "responses": [
			 {
			   "aggregations": {
				 "2": {
				   "buckets": [
					 {
					   "3": {
						 "buckets": [{ "doc_count": 1, "key": 1000 }, { "doc_count": 3, "key": 2000 }]
					   },
					   "doc_count": 4,
					   "key": "server1"
					 },
					 {
					   "3": {
						 "buckets": [{ "doc_count": 2, "key": 1000 }, { "doc_count": 8, "key": 2000 }]
					   },
					   "doc_count": 10,
					   "key": "server2"
					 },
					 {
					   "3": {
						 "buckets": [{ "doc_count": 2, "key": 1000 }, { "doc_count": 8, "key": 2000 }]
					   },
					   "doc_count": 10,
					   "key": 0
					 }
				   ]
				 }
			   }
			 }
		   ]
		}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		assert.Len(t, queryRes.Frames, 3)

		seriesOne := queryRes.Frames[0]
		assert.Equal(t, "server1 Count and {{not_exist}} server1", seriesOne.Name)
		require.Len(t, seriesOne.Fields, 2)
		require.Equal(t, 2, seriesOne.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesOne.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesOne.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesOne.Fields[1].Len())
		assert.EqualValues(t, 1, *seriesOne.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 3, *seriesOne.Fields[1].At(1).(*float64))

		seriesTwo := queryRes.Frames[1]
		assert.Equal(t, "server2 Count and {{not_exist}} server2", seriesTwo.Name)
		require.Len(t, seriesTwo.Fields, 2)
		require.Equal(t, 2, seriesTwo.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesTwo.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesTwo.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesTwo.Fields[1].Len())
		assert.EqualValues(t, 2, *seriesTwo.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 8, *seriesTwo.Fields[1].At(1).(*float64))

		seriesThree := queryRes.Frames[2]
		assert.Equal(t, "0 Count and {{not_exist}} 0", seriesThree.Name)
		require.Len(t, seriesThree.Fields, 2)
		require.Equal(t, 2, seriesThree.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesThree.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesThree.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesThree.Fields[1].Len())
		assert.EqualValues(t, 2, *seriesThree.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 8, *seriesThree.Fields[1].At(1).(*float64))
	})

	// TODO: this test will require some conversion of tables to data frames, original work in Elasticsearch https://github.com/grafana/grafana/pull/34710; https://github.com/grafana/opensearch-datasource/issues/175
	//t.Run("Histogram response", func(t *testing.T) {
	//	targets := map[string]string{
	//		"A": `{
	//				"timeField": "@timestamp",
	//				"metrics": [{ "type": "count", "id": "1" }],
	//	 "bucketAggs": [{ "type": "histogram", "field": "bytes", "id": "3" }]
	//			}`,
	//	}
	//	response := `{
	//	   "responses": [
	//		 {
	//		   "aggregations": {
	//			 "3": {
	//			   "buckets": [{ "doc_count": 1, "key": 1000 }, { "doc_count": 3, "key": 2000 }, { "doc_count": 2, "key": 3000 }]
	//			 }
	//		   }
	//		 }
	//	   ]
	//			}`
	//	rp, err := newResponseParserForTest(targets, response)
	//	assert.Nil(t, err)
	//	result, err := rp.getTimeSeries()
	//	assert.Nil(t, err)
	//	require.Len(t, result.Responses, 1)
	//
	//	queryRes := result.Responses["A"]
	//	assert.NotNil(t, queryRes)
	//	assert.Len(t, queryRes.Frames, 1)
	//So(queryRes.Tables, ShouldHaveLength, 1)
	//
	//rows := queryRes.Tables[0].Rows
	//So(rows, ShouldHaveLength, 3)
	//cols := queryRes.Tables[0].Columns
	//So(cols, ShouldHaveLength, 2)
	//
	//So(cols[0].Text, ShouldEqual, "bytes")
	//So(cols[1].Text, ShouldEqual, "Count")
	//
	//So(rows[0][0].(null.Float).Float64, ShouldEqual, 1000)
	//So(rows[0][1].(null.Float).Float64, ShouldEqual, 1)
	//So(rows[1][0].(null.Float).Float64, ShouldEqual, 2000)
	//So(rows[1][1].(null.Float).Float64, ShouldEqual, 3)
	//So(rows[2][0].(null.Float).Float64, ShouldEqual, 3000)
	//So(rows[2][1].(null.Float).Float64, ShouldEqual, 2)
	//})

	t.Run("With two filters agg", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
				"timeField": "@timestamp",
				"metrics": [{ "type": "count", "id": "1" }],
	 "bucketAggs": [
					{
						"type": "filters",
						"id": "2",
						"settings": {
							"filters": [{ "query": "@metric:cpu" }, { "query": "@metric:logins.count" }]
						}
					},
					{ "type": "date_histogram", "field": "@timestamp", "id": "3" }
				]
			}`,
		}
		response := `{
		   "responses": [
			 {
			   "aggregations": {
				 "2": {
				   "buckets": {
					 "@metric:cpu": {
					   "3": {
						 "buckets": [{ "doc_count": 1, "key": 1000 }, { "doc_count": 3, "key": 2000 }]
					   }
					 },
					 "@metric:logins.count": {
					   "3": {
						 "buckets": [{ "doc_count": 2, "key": 1000 }, { "doc_count": 8, "key": 2000 }]
					   }
					 }
				   }
				 }
			   }
			 }
		   ]
				}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		assert.Len(t, queryRes.Frames, 2)

		seriesOne := queryRes.Frames[0]
		assert.Equal(t, "@metric:cpu", seriesOne.Name)
		require.Len(t, seriesOne.Fields, 2)
		require.Equal(t, 2, seriesOne.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesOne.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesOne.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesOne.Fields[1].Len())
		assert.EqualValues(t, 1, *seriesOne.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 3, *seriesOne.Fields[1].At(1).(*float64))

		seriesTwo := queryRes.Frames[1]
		assert.Equal(t, "@metric:logins.count", seriesTwo.Name)
		require.Len(t, seriesTwo.Fields, 2)
		require.Equal(t, 2, seriesTwo.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesTwo.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesTwo.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesTwo.Fields[1].Len())
		assert.EqualValues(t, 2, *seriesTwo.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 8, *seriesTwo.Fields[1].At(1).(*float64))
	})

	t.Run("With dropfirst and last aggregation", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
					"timeField": "@timestamp",
					"metrics": [{ "type": "avg", "id": "1" }, { "type": "count" }],
		 "bucketAggs": [
						{
							"type": "date_histogram",
							"field": "@timestamp",
							"id": "2",
							"settings": { "trimEdges": 1 }
						}
					]
				}`,
		}
		response := `{
		   "responses": [
			 {
			   "aggregations": {
				 "2": {
				   "buckets": [
					 {
					   "1": { "value": 11 },
					   "key": 1000,
					   "doc_count": 369
					 },
					 {
					   "1": { "value": 22 },
					   "key": 2000,
					   "doc_count": 200
					 },
					 {
					   "1": { "value": 33 },
					   "key": 3000,
					   "doc_count": 200
					 }
				   ]
				 }
			   }
			 }
		   ]
				}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		assert.Len(t, queryRes.Frames, 2)

		seriesOne := queryRes.Frames[0]
		assert.Equal(t, "Average", seriesOne.Name)
		require.Len(t, seriesOne.Fields, 2)
		require.Equal(t, 1, seriesOne.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesOne.Fields[0].At(0).(*time.Time))
		require.Equal(t, 1, seriesOne.Fields[1].Len())
		assert.EqualValues(t, 22, *seriesOne.Fields[1].At(0).(*float64))

		seriesTwo := queryRes.Frames[1]
		assert.Equal(t, "Count", seriesTwo.Name)
		require.Len(t, seriesTwo.Fields, 2)
		require.Equal(t, 1, seriesTwo.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesTwo.Fields[0].At(0).(*time.Time))
		require.Equal(t, 1, seriesTwo.Fields[1].Len())
		assert.EqualValues(t, 200, *seriesTwo.Fields[1].At(0).(*float64))
	})

	// TODO: similar to above, this test will require some conversion of tables to data frames, original work in Elasticsearch https://github.com/grafana/grafana/pull/34710; https://github.com/grafana/opensearch-datasource/issues/175
	//t.Run("No group by time", func(t *testing.T) {
	//	targets := map[string]string{
	//		"A": `{
	//				"timeField": "@timestamp",
	//				"metrics": [{ "type": "avg", "id": "1" }, { "type": "count" }],
	//	 "bucketAggs": [{ "type": "terms", "field": "host", "id": "2" }]
	//			}`,
	//	}
	//	response := `{
	//	   "responses": [
	//		 {
	//		   "aggregations": {
	//			 "2": {
	//			   "buckets": [
	//				 {
	//				   "1": { "value": 1000 },
	//				   "key": "server-1",
	//				   "doc_count": 369
	//				 },
	//				 {
	//				   "1": { "value": 2000 },
	//				   "key": "server-2",
	//				   "doc_count": 200
	//				 }
	//			   ]
	//			 }
	//		   }
	//		 }
	//	   ]
	//			}`
	//	rp, err := newResponseParserForTest(targets, response)
	//	assert.Nil(t, err)
	//	result, err := rp.getTimeSeries()
	//	assert.Nil(t, err)
	//	require.Len(t, result.Responses, 1)
	//
	//	queryRes := result.Responses["A"]
	//	assert.NotNil(t, queryRes)
	//	So(queryRes.Tables, ShouldHaveLength, 1)
	//
	//	rows := queryRes.Tables[0].Rows
	//	So(rows, ShouldHaveLength, 2)
	//	cols := queryRes.Tables[0].Columns
	//	So(cols, ShouldHaveLength, 3)
	//
	//	So(cols[0].Text, ShouldEqual, "host")
	//	So(cols[1].Text, ShouldEqual, "Average")
	//	So(cols[2].Text, ShouldEqual, "Count")
	//
	//	So(rows[0][0].(string), ShouldEqual, "server-1")
	//	So(rows[0][1].(null.Float).Float64, ShouldEqual, 1000)
	//	So(rows[0][2].(null.Float).Float64, ShouldEqual, 369)
	//	So(rows[1][0].(string), ShouldEqual, "server-2")
	//	So(rows[1][1].(null.Float).Float64, ShouldEqual, 2000)
	//	So(rows[1][2].(null.Float).Float64, ShouldEqual, 200)
	//})
	//
	// TODO: similar to above, this test will require some conversion of tables to data frames, original work in Elasticsearch https://github.com/grafana/grafana/pull/34710; https://github.com/grafana/opensearch-datasource/issues/175
	//t.Run("Multiple metrics of same type", func(t *testing.T) {
	//	targets := map[string]string{
	//		"A": `{
	//					"timeField": "@timestamp",
	//					"metrics": [{ "type": "avg", "field": "test", "id": "1" }, { "type": "avg", "field": "test2", "id": "2" }],
	//		 "bucketAggs": [{ "type": "terms", "field": "host", "id": "2" }]
	//				}`,
	//	}
	//	response := `{
	//	   "responses": [
	//		 {
	//		   "aggregations": {
	//			 "2": {
	//			   "buckets": [
	//				 {
	//				   "1": { "value": 1000 },
	//				   "2": { "value": 3000 },
	//				   "key": "server-1",
	//				   "doc_count": 369
	//				 }
	//			   ]
	//			 }
	//		   }
	//		 }
	//	   ]
	//			}`
	//	rp, err := newResponseParserForTest(targets, response)
	//	assert.Nil(t, err)
	//	result, err := rp.getTimeSeries()
	//	assert.Nil(t, err)
	//	require.Len(t, result.Responses, 1)
	//
	//	queryRes := result.Responses["A"]
	//	assert.NotNil(t, queryRes)
	//	So(queryRes.Tables, ShouldHaveLength, 1)
	//
	//	rows := queryRes.Tables[0].Rows
	//	So(rows, ShouldHaveLength, 1)
	//	cols := queryRes.Tables[0].Columns
	//	So(cols, ShouldHaveLength, 3)
	//
	//	So(cols[0].Text, ShouldEqual, "host")
	//	So(cols[1].Text, ShouldEqual, "Average test")
	//	So(cols[2].Text, ShouldEqual, "Average test2")
	//
	//	So(rows[0][0].(string), ShouldEqual, "server-1")
	//	So(rows[0][1].(null.Float).Float64, ShouldEqual, 1000)
	//	So(rows[0][2].(null.Float).Float64, ShouldEqual, 3000)
	//})

	t.Run("With bucket_script", func(t *testing.T) {
		targets := map[string]string{
			"A": `{
				"timeField": "@timestamp",
				"metrics": [
					{ "id": "1", "type": "sum", "field": "@value" },
					{ "id": "3", "type": "max", "field": "@value" },
					{
					 "id": "4",
					 "field": "select field",
					 "pipelineVariables": [{ "name": "var1", "pipelineAgg": "1" }, { "name": "var2", "pipelineAgg": "3" }],
					 "settings": { "script": "params.var1 * params.var2" },
					 "type": "bucket_script"
					}
				],
		 "bucketAggs": [{ "type": "date_histogram", "field": "@timestamp", "id": "2" }]
				}`,
		}
		response := `{
		   "responses": [
			 {
			   "aggregations": {
				 "2": {
				   "buckets": [
					 {
					   "1": { "value": 2 },
					   "3": { "value": 3 },
					   "4": { "value": 6 },
					   "doc_count": 60,
					   "key": 1000
					 },
					 {
					   "1": { "value": 3 },
					   "3": { "value": 4 },
					   "4": { "value": 12 },
					   "doc_count": 60,
					   "key": 2000
					 }
				   ]
				 }
			   }
			 }
		   ]
				}`
		rp, err := newResponseParserForTest(targets, response)
		assert.Nil(t, err)
		result, err := rp.getTimeSeries()
		assert.Nil(t, err)
		require.Len(t, result.Responses, 1)

		queryRes := result.Responses["A"]
		assert.NotNil(t, queryRes)
		assert.Len(t, queryRes.Frames, 3)
		seriesOne := queryRes.Frames[0]
		assert.Equal(t, "Sum @value", seriesOne.Name)
		require.Len(t, seriesOne.Fields, 2)
		require.Equal(t, 2, seriesOne.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesOne.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesOne.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesOne.Fields[1].Len())
		assert.EqualValues(t, 2, *seriesOne.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 3, *seriesOne.Fields[1].At(1).(*float64))

		seriesTwo := queryRes.Frames[1]
		assert.Equal(t, "Max @value", seriesTwo.Name)
		require.Len(t, seriesTwo.Fields, 2)
		require.Equal(t, 2, seriesTwo.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesTwo.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesTwo.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesTwo.Fields[1].Len())
		assert.EqualValues(t, 3, *seriesTwo.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 4, *seriesTwo.Fields[1].At(1).(*float64))

		seriesThree := queryRes.Frames[2]
		assert.Equal(t, "Sum @value * Max @value", seriesThree.Name)
		require.Len(t, seriesThree.Fields, 2)
		require.Equal(t, 2, seriesThree.Fields[0].Len())
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC), *seriesThree.Fields[0].At(0).(*time.Time))
		assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 2, 0, time.UTC), *seriesThree.Fields[0].At(1).(*time.Time))
		require.Equal(t, 2, seriesThree.Fields[1].Len())
		assert.EqualValues(t, 6, *seriesThree.Fields[1].At(0).(*float64))
		assert.EqualValues(t, 12, *seriesThree.Fields[1].At(1).(*float64))
	})

	// TODO: similar to above, this test will require some conversion of tables to data frames, original work in Elasticsearch https://github.com/grafana/grafana/pull/34710; https://github.com/grafana/opensearch-datasource/issues/175
	//t.Run("Terms with two bucket_script", func(t *testing.T) {
	//	targets := map[string]string{
	//		"A": `{
	//			"timeField": "@timestamp",
	//			"metrics": [
	//				{ "id": "1", "type": "sum", "field": "@value" },
	//			{ "id": "3", "type": "max", "field": "@value" },
	//			{
	//					"id": "4",
	//					"field": "select field",
	//					"pipelineVariables": [{ "name": "var1", "pipelineAgg": "1" }, { "name": "var2", "pipelineAgg": "3" }],
	//					"settings": { "script": "params.var1 * params.var2" },
	//					"type": "bucket_script"
	//				},
	//			{
	//					"id": "5",
	//					"field": "select field",
	//					"pipelineVariables": [{ "name": "var1", "pipelineAgg": "1" }, { "name": "var2", "pipelineAgg": "3" }],
	//					"settings": { "script": "params.var1 * params.var2 * 2" },
	//					"type": "bucket_script"
	//			  }
	//			],
	// "bucketAggs": [{ "type": "terms", "field": "@timestamp", "id": "2" }]
	//		}`,
	//	}
	//	response := `{
	//		"responses": [
	//			{
	//				"aggregations": {
	//				"2": {
	//					"buckets": [
	//					{
	//						"1": { "value": 2 },
	//						"3": { "value": 3 },
	//						"4": { "value": 6 },
	//						"5": { "value": 24 },
	//						"doc_count": 60,
	//						"key": 1000
	//					},
	//					{
	//						"1": { "value": 3 },
	//						"3": { "value": 4 },
	//						"4": { "value": 12 },
	//						"5": { "value": 48 },
	//						"doc_count": 60,
	//						"key": 2000
	//					}
	//					]
	//				}
	//				}
	//			}
	//		]
	//	}`
	//	rp, err := newResponseParserForTest(targets, response)
	//	assert.Nil(t, err)
	//	result, err := rp.getTimeSeries()
	//	assert.Nil(t, err)
	//	require.Len(t, result.Responses, 1)
	//	queryRes := result.Responses["A"]
	//	assert.NotNil(t, queryRes)
	//So(queryRes.Tables[0].Rows, ShouldHaveLength, 2)
	//So(queryRes.Tables[0].Columns[1].Text, ShouldEqual, "Sum")
	//So(queryRes.Tables[0].Columns[2].Text, ShouldEqual, "Max")
	//So(queryRes.Tables[0].Columns[3].Text, ShouldEqual, "params.var1 * params.var2")
	//So(queryRes.Tables[0].Columns[4].Text, ShouldEqual, "params.var1 * params.var2 * 2")
	//So(queryRes.Tables[0].Rows[0][1].(null.Float).Float64, ShouldEqual, 2)
	//So(queryRes.Tables[0].Rows[0][2].(null.Float).Float64, ShouldEqual, 3)
	//So(queryRes.Tables[0].Rows[0][3].(null.Float).Float64, ShouldEqual, 6)
	//So(queryRes.Tables[0].Rows[0][4].(null.Float).Float64, ShouldEqual, 24)
	//So(queryRes.Tables[0].Rows[1][1].(null.Float).Float64, ShouldEqual, 3)
	//So(queryRes.Tables[0].Rows[1][2].(null.Float).Float64, ShouldEqual, 4)
	//So(queryRes.Tables[0].Rows[1][3].(null.Float).Float64, ShouldEqual, 12)
	//So(queryRes.Tables[0].Rows[1][4].(null.Float).Float64, ShouldEqual, 48)
	//})

	//t.Run("Raw documents query", func(t *testing.T) {
	//	targets := map[string]string{
	//		"A": `{
	//						"timeField": "@timestamp",
	//						"metrics": [{ "type": "raw_document", "id": "1" }]
	//					}`,
	//	}
	//	response := `{
	//			    "responses": [
	//			      {
	//			        "hits": {
	//			          "total": 100,
	//			          "hits": [
	//			            {
	//			              "_id": "1",
	//			              "_type": "type",
	//			              "_index": "index",
	//			              "_source": { "sourceProp": "asd" },
	//			              "fields": { "fieldProp": "field" }
	//			            },
	//			            {
	//			              "_source": { "sourceProp": "asd2" },
	//			              "fields": { "fieldProp": "field2" }
	//			            }
	//			          ]
	//			        }
	//			      }
	//			    ]
	//				}`
	//	rp, err := newResponseParserForTest(targets, response)
	//	assert.Nil(t, err)
	//	result, err := rp.getTimeSeries()
	//	assert.Nil(t, err)
	//	require.Len(t, result.Responses, 1)
	//
	//	queryRes := result.Responses["A"]
	//	assert.NotNil(t, queryRes)
	//So(queryRes.Tables, ShouldHaveLength, 1)
	//
	//rows := queryRes.Tables[0].Rows
	//So(rows, ShouldHaveLength, 1)
	//cols := queryRes.Tables[0].Columns
	//So(cols, ShouldHaveLength, 3)
	//
	//So(cols[0].Text, ShouldEqual, "host")
	//So(cols[1].Text, ShouldEqual, "Average test")
	//So(cols[2].Text, ShouldEqual, "Average test2")
	//
	//So(rows[0][0].(string), ShouldEqual, "server-1")
	//So(rows[0][1].(null.Float).Float64, ShouldEqual, 1000)
	//So(rows[0][2].(null.Float).Float64, ShouldEqual, 3000)
	//})
}

func newResponseParserForTest(tsdbQueries map[string]string, responseBody string) (*responseParser, error) {
	from := time.Date(2018, 5, 15, 17, 50, 0, 0, time.UTC)
	to := time.Date(2018, 5, 15, 17, 55, 0, 0, time.UTC)
	tsdbQuery := &backend.QueryDataRequest{
		Queries: []backend.DataQuery{},
	}

	for refID, tsdbQueryBody := range tsdbQueries {
		tsdbQuery.Queries = append(tsdbQuery.Queries, backend.DataQuery{
			JSON:  []byte(tsdbQueryBody),
			RefID: refID,
			TimeRange: backend.TimeRange{
				From: from,
				To:   to,
			},
		})
	}

	var response client.MultiSearchResponse
	err := json.Unmarshal([]byte(responseBody), &response)
	if err != nil {
		return nil, err
	}

	tsQueryParser := newTimeSeriesQueryParser()
	queries, err := tsQueryParser.parse(tsdbQuery)
	if err != nil {
		return nil, err
	}

	return newResponseParser(response.Responses, queries, nil), nil
}
