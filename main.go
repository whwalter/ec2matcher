package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	//	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {

	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ec2Client := ec2.New(awsSession)

	primaryType := aws.String("r5.8xlarge")
	typeInput := ec2.DescribeInstanceTypesInput{
		DryRun: aws.Bool(false),
		InstanceTypes: []*string{primaryType},
	}

	err := typeInput.Validate()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		panic("aaaahhhh")
	}
	dit, err := ec2Client.DescribeInstanceTypes(&typeInput)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		panic("aaaaahhhh")
	}

	prime := dit.InstanceTypes[0]
	fmt.Println(prime)
	matchers := []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("memory-info.size-in-mib"),
				Values: []*string{
					aws.String(fmt.Sprint(*prime.MemoryInfo.SizeInMiB)),
					},
		  	 },
			&ec2.Filter{
				Name: aws.String("vcpu-info.default-vcpus"),
				Values: []*string{
					aws.String(fmt.Sprint(*prime.VCpuInfo.DefaultVCpus)),
					},
		  	 },
			&ec2.Filter{
				Name: aws.String("bare-metal"),
				Values: []*string{
					aws.String("false"),
					},
		  	 },
			&ec2.Filter{
				Name: aws.String("hypervisor"),
				Values: []*string{
					prime.Hypervisor,
					},
		  	 },
		   }
/*
	matchers := []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("memory-info.size-in-mib"),
				Values: []*string{
					aws.String(fmt.Sprint(*prime.MemoryInfo.SizeInMiB)),
					},
		  	 },
			&ec2.Filter{
				Name: aws.String("vcpu-info.default-vcpus"),
				Values: []*string{
					aws.String(fmt.Sprint(*prime.VCpuInfo.DefaultVCpus)),
					},
		  	 },
		   }

*/
	typeInput = ec2.DescribeInstanceTypesInput{
		DryRun: aws.Bool(false),
		Filters: matchers,
		MaxResults: aws.Int64(5),
	}
	err = typeInput.Validate()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		panic("Failed to validate matching query")
	}
	var its []*ec2.InstanceTypeInfo
	err = ec2Client.DescribeInstanceTypesPages(
		&typeInput,
		func(page *ec2.DescribeInstanceTypesOutput, lastPage bool) bool{
			for _,v := range page.InstanceTypes {
				its = append(its, v)
			}
			if lastPage {
				return false
			}
			return true
		})
	if err != nil {
		fmt.Printf("error: %v\n", err)
		panic("Failed to return matching query")
	}
	for _, v := range its {
		fmt.Printf("Name: %s\nRam: %d\nVcpus: %d\n", *v.InstanceType, *v.MemoryInfo.SizeInMiB, *v.VCpuInfo.DefaultVCpus)
	}
/*
	fmt.Printf("Number of matches: %d\n\n", len(matchedInstances.InstanceTypes))
	for _,v := range matchedInstances.InstanceTypes {
		fmt.Printf("Name: %s\nRam: %d\nVcpus: %d\n", *v.InstanceType, *v.MemoryInfo.SizeInMiB, *v.VCpuInfo.DefaultVCpus)
	}

	if len(matchedInstances.InstanceTypes) == 0 {
		fmt.Println("No matches for filters")
	}

	if matchedInstances.NextToken != nil {
		fmt.Printf(*matchedInstances.NextToken)
	}
*/
}

func filterInstanceTypes(describer ec2.DescribeInstanceTypesInput) (instanceTypes []*ec2.InstanceTypeInfo, e error) {
	var its []*ec2.InstanceTypeInfo
	err = ec2Client.DescribeInstanceTypesPages(
		&typeInput,
		func(page *ec2.DescribeInstanceTypesOutput, lastPage bool) bool{
			for _,v := range page.InstanceTypes {
				its = append(its, v)
			}
			if lastPage {
				return false
			}
			return true
		})
	if err != nil {
		return its,err
	}
	return its, nil
}
