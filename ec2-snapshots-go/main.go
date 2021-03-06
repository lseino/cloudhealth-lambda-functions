package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func handler(request map[string]interface{}) error {
	// Get the ARNs from the CloudHealth message
	events := request["resource_arns"].([]interface{})

	// Create a new session without AWS credentials.
	// This means the Lambda function must have privileges to access EC2
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	// Create a client to the EC2 service
	ec2Client := ec2.New(awsSession)

	// Loop over the ARNs and if they are EC2 instances, try to create a snapshot
	for _, event := range events {
		arn := event.(string)
		elem := strings.Split(arn, ":")
		if strings.Contains(arn, ":instance:") {
			// Get the BlockDeviceMapping attribute of the EC2 instance
			instanceAttributesInput := &ec2.DescribeInstanceAttributeInput{
				Attribute:  aws.String("blockDeviceMapping"),
				InstanceId: aws.String(elem[6]),
			}

			instanceAttributesOutput, err := ec2Client.DescribeInstanceAttribute(instanceAttributesInput)
			if err != nil {
				fmt.Printf("error getting attributes of %s: %s\n", elem, err.Error())
				continue
			}

			// Get the volumeID of the first BlockDevice
			volumeID := instanceAttributesOutput.BlockDeviceMappings[0].Ebs.VolumeId

			// Create a snapshot request
			snapshotRequestInput := &ec2.CreateSnapshotInput{
				DryRun:   aws.Bool(false),
				VolumeId: volumeID,
			}

			snapshotRequest, snapshotRequestOutput := ec2Client.CreateSnapshotRequest(snapshotRequestInput)

			// Execute the snapshot request
			err = snapshotRequest.Send()
			if err != nil {
				fmt.Printf("error occurred while trying to take snapshot of %s: %s\n", elem, err.Error())
				continue
			}
			fmt.Printf("current status for %s: %s\n", elem, *snapshotRequestOutput.State)
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
