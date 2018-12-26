package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

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

func main() {
	now := time.Now()
	startTime := now.Unix()
	endTime := now.Add(-5 * time.Minute).Unix()
	limit := 10
	logGroupName := "/aws/lambda/hoge"
	query := "fields @timestamp, @message | sort @timestamp desc"

	cwl := NewAwsCloudWatchClient()
	inputStartQuery, err := NewStartQueryInput(startTime, endTime, int64(limit), logGroupName, query)
	if err != nil {
		panic(err)
	}
	startQueryOutput, err := cwl.StartQuery(inputStartQuery)
	if err != nil {
		panic(err)
	}
	resultsOutput, err := getQueryResultsUntilCompleate(cwl, *startQueryOutput.QueryId, limit)
	if err != nil {
		panic(err)
	}
	for _, rs := range resultsOutput.Results {
		for _, v := range rs {
			fmt.Printf("field=%s, value=%s", *v.Field, *v.Value)
		}
	}
}

func getQueryResultsUntilCompleate(cwl *cloudwatchlogs.CloudWatchLogs, queryId string, limit int) (*cloudwatchlogs.GetQueryResultsOutput, error) {
	getQueryResultInput := &cloudwatchlogs.GetQueryResultsInput{}
	getQueryResultInput.SetQueryId(queryId)
	fmt.Println(queryId)
	for {
		getQueryResultOutput, err := cwl.GetQueryResults(getQueryResultInput)
		if err != nil {
			return nil, err
		}
		time.Sleep(5 * time.Second)
		switch *getQueryResultOutput.Status {
		case "Running", "Scheduled":
			if len(getQueryResultOutput.Results) < limit {
				continue
			}
			// FIXME クエリ止めたいが止めれない
			// fmt.Println(queryId)
			// stopQueryInput := &cloudwatchlogs.StopQueryInput{}
			// stopQueryInput.SetQueryId(queryId)
			// stopResult, err := cwl.StopQuery(stopQueryInput)
			// if err != nil {
			// 	return nil, fmt.Errorf("あかんなんか死んだわ: error=%s status=%v", err.Error(), stopResult)
			// }
			return getQueryResultOutput, nil
		case "Failed", "Cancelled":
			return nil, fmt.Errorf("あかんなんか死んだわ: %s", getQueryResultOutput.String())
		case "Complete":
			return getQueryResultOutput, nil
		default:
			return nil, fmt.Errorf("あかんなんか死んだわ: %s", getQueryResultOutput.String())
		}
	}
}
