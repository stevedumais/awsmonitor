package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
)

// EC2Instance is a subset of available EC2 metadata
// pulled for metadata
type EC2Instance struct {
	ID    string
	Type  string
	State string
	IP    string
	Name  string
}

// getNameFromTags is a helper to grab the instance name from the array of tags
// This is of course dependent on ec2 instances being given a name KV pair
func getNameFromTags(tags []*ec2.Tag) string {
	name := ""
	for _, tag := range tags {
		if *tag.Key == "Name" {
			name = *tag.Value
			break
		}
	}
	return name
}

// yStart gets yesterday start time
func yStart(t time.Time) time.Time {
	year, month, day := t.Date()
	date := time.Date(year, month, day, 0, 0, 0, 0, t.Location())
	date = date.Add(-24 * time.Hour)
	return date
}

// yStart gets yesterday end time
func yEnd(t time.Time) time.Time {
	year, month, day := t.Date()
	date := time.Date(year, month, day, 23, 59, 59, 999999999, t.Location())
	date = date.Add(-24 * time.Hour)
	return date
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists AWS EC2 instance data",
	Long: `Grabs a lists of EC2 instances, then pulls network data
	 so that you can determine if any of these could be throttled down.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		ny, err := time.LoadLocation("America/New_York")
		if err != nil {
			log.Fatal(err)
		}
		yesterdayStart := yStart(time.Now().In(ny))
		yesterdayEnd := yEnd(time.Now().In(ny))

		sess := session.Must(session.NewSessionWithOptions(session.Options{
			Config: aws.Config{Region: aws.String("us-east-1")},
		}))

		ec2Svc := ec2.New(sess)
		cSvc := cloudwatch.New(sess)
		instances, err := ec2Svc.DescribeInstances(nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Name,Type,State,ID,IP,NetworkIn,NetworkOut")

		for _, reservations := range instances.Reservations {
			for _, instance := range reservations.Instances {
				result, err := cSvc.GetMetricData(&cloudwatch.GetMetricDataInput{
					StartTime: &yesterdayStart,
					EndTime:   &yesterdayEnd,
					MetricDataQueries: []*cloudwatch.MetricDataQuery{
						&cloudwatch.MetricDataQuery{
							Id: aws.String("cw1"),
							MetricStat: &cloudwatch.MetricStat{
								Stat:   aws.String("Sum"),
								Period: aws.Int64(86400),
								Metric: &cloudwatch.Metric{
									Namespace:  aws.String("AWS/EC2"),
									MetricName: aws.String("NetworkIn"),
									Dimensions: []*cloudwatch.Dimension{
										&cloudwatch.Dimension{
											Name:  aws.String("InstanceId"),
											Value: aws.String(*instance.InstanceId),
										},
									},
								},
							},
						},
						&cloudwatch.MetricDataQuery{
							Id: aws.String("cw2"),
							MetricStat: &cloudwatch.MetricStat{
								Stat:   aws.String("Sum"),
								Period: aws.Int64(86400),
								Metric: &cloudwatch.Metric{
									Namespace:  aws.String("AWS/EC2"),
									MetricName: aws.String("NetworkOut"),
									Dimensions: []*cloudwatch.Dimension{
										&cloudwatch.Dimension{
											Name:  aws.String("InstanceId"),
											Value: aws.String(*instance.InstanceId),
										},
									},
								},
							},
						},
					},
				})
				if err != nil {
					log.Fatal(err)
				}
				var (
					netIn  float64
					netOut float64
				)

				for _, res := range result.MetricDataResults {
					if *res.Id == "cw1" && len(res.Values) > 0 {
						netIn = *res.Values[0]
					}
					if *res.Id == "cw2" && len(res.Values) > 0 {
						netOut = *res.Values[0]
					}
				}

				fmt.Printf("%v,%v,%v,%v,%v,%v,%v\n",
					getNameFromTags(instance.Tags),
					*instance.InstanceType,
					*instance.State.Name,
					*instance.InstanceId,
					*instance.PrivateIpAddress,
					netIn,
					netOut,
				)

			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
