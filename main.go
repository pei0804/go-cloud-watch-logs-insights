package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

var (
	startTimeStr = flag.String("s", "2018-12-25", "start time")
	endTimeStr   = flag.String("e", "2018-12-26", "end time")
	logGroupName = flag.String("n", "/aws/lambda/hoge", "log group name")
	limit        = flag.Int("l", 10, "limit")
	query        = flag.String("q", "fields @timestamp, @message | sort @timestamp desc", "query")
)

const DateLayout = "2006-01-02"

func main() {
	flag.Parse()
	startTime, err := time.Parse(DateLayout, *startTimeStr)
	if err != nil {
		panic(err)
	}
	endTime, err := time.Parse(DateLayout, *endTimeStr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("start time = %s\n", startTime)
	fmt.Printf("end time = %s\n", endTime)
	fmt.Printf("limit = %d\n", *limit)
	fmt.Printf("log group name = %s\n", *logGroupName)
	fmt.Printf("query = %s\n", *query)

	cwl := NewAwsCloudWatchClient()
	inputStartQuery, err := NewStartQueryInput(startTime.Unix(), endTime.Unix(), int64(*limit), *logGroupName, *query)
	if err != nil {
		panic(err)
	}
	startQueryOutput, err := cwl.StartQuery(inputStartQuery)
	if err != nil {
		panic(err)
	}
	resultsOutput, err := getQueryResultsUntilCompleate(cwl, *startQueryOutput.QueryId, *limit)
	if err != nil {
		panic(err)
	}
	for _, rs := range resultsOutput.Results {
		for _, v := range rs {
			fmt.Printf("field=%s, value=%s\n", *v.Field, *v.Value)
		}
	}
}

func NewAwsCloudWatchClient() *cloudwatchlogs.CloudWatchLogs {
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(os.Getenv("AccessKeyID"), os.Getenv("SecretAccessKey"), os.Getenv("SessionToken"))
	return cloudwatchlogs.New(
		sess,
		aws.NewConfig().WithRegion("ap-northeast-1").WithCredentials(creds),
	)
}

func NewStartQueryInput(startTime int64, endTime int64, limit int64, logGroupName string, query string) (*cloudwatchlogs.StartQueryInput, error) {
	i := &cloudwatchlogs.StartQueryInput{}
	i.SetLimit(limit)
	i.SetStartTime(startTime)
	i.SetEndTime(endTime)
	i.SetLogGroupName(logGroupName)
	i.SetQueryString(query)
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return i, nil
}

func getQueryResultsUntilCompleate(cwl *cloudwatchlogs.CloudWatchLogs, queryId string, limit int) (*cloudwatchlogs.GetQueryResultsOutput, error) {
	getQueryResultInput := &cloudwatchlogs.GetQueryResultsInput{}
	getQueryResultInput.SetQueryId(queryId)
	for {
		getQueryResultOutput, err := cwl.GetQueryResults(getQueryResultInput)
		if err != nil {
			return nil, err
		}
		time.Sleep(5 * time.Second)
		switch *getQueryResultOutput.Status {
		case "Running":
			if len(getQueryResultOutput.Results) < limit {
				continue
			}
			stopQueryInput := &cloudwatchlogs.StopQueryInput{}
			stopQueryInput.SetQueryId(queryId)
			stopResult, err := cwl.StopQuery(stopQueryInput)
			if err != nil {
				return nil, fmt.Errorf("stop query error=%s status=%v", err.Error(), stopResult)
			}
			return getQueryResultOutput, nil
		case "Scheduled":
			continue
		case "Failed":
			return nil, errors.New("job failed")
		case "Cancelled":
			return nil, errors.New("job cancelled")
		case "Complete":
			return getQueryResultOutput, nil
		default:
			return nil, fmt.Errorf("unknown status: %s", *getQueryResultOutput.Status)
		}
	}
}
