package main

import (
	"fmt"
//	"strings"

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
	its, err := filterInstanceTypes(typeInput, ec2Client)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	var ots []string
	for k, v := range its {
		fmt.Printf("Name: %s\nRam: %d\nVcpus: %d\n", *v.InstanceType, *v.MemoryInfo.SizeInMiB, *v.VCpuInfo.DefaultVCpus)
		ots = append(ots, k)
	}

	offeringFilters := []*ec2.Filter{
				&ec2.Filter{
					Name: aws.String("location"),
					Values: aws.StringSlice([]string{"us-east-1c","us-east-1b"}),
				},
				&ec2.Filter{
					Name: aws.String("instance-type"),
					Values: aws.StringSlice(ots),
				},
			}
	fmt.Print(offeringFilters)
	offeringInput := ec2.DescribeInstanceTypeOfferingsInput{
				Filters: offeringFilters,
				LocationType: aws.String("availability-zone"),
				MaxResults: aws.Int64(5),
			}
	err = offeringInput.Validate()
	if err != nil {
		fmt.Printf("Failed to validate offering filters: %v\n", err)
		panic("AAAAAAHHHHH")
	}

	typesInZones, err := filterInstanceTypeOfferings(offeringInput, ec2Client)
	if err != nil {
		fmt.Printf("Failed to retrieve offerings: %v\n", err)
		panic("AAAAAAHHHHH")
	}
	fmt.Println(typesInZones)
}

func filterInstanceTypes(describer ec2.DescribeInstanceTypesInput, client *ec2.EC2) (instanceTypes map[string]*ec2.InstanceTypeInfo, e error) {
	its := map[string]*ec2.InstanceTypeInfo{}
	err := client.DescribeInstanceTypesPages(
		&describer,
		func(page *ec2.DescribeInstanceTypesOutput, lastPage bool) bool{
			for _,v := range page.InstanceTypes {
				its[aws.StringValue(v.InstanceType)] = v
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

func filterInstanceTypeOfferings(describer ec2.DescribeInstanceTypeOfferingsInput, client *ec2.EC2) (instanceTypeOfferings []*ec2.InstanceTypeOffering, e error) {
	var ito []*ec2.InstanceTypeOffering
	err := client.DescribeInstanceTypeOfferingsPages(
		&describer,
		func(page *ec2.DescribeInstanceTypeOfferingsOutput, lastPage bool) bool{
//			fmt.Println(len(page.InstanceTypeOfferings))
			for _,v := range page.InstanceTypeOfferings {
				ito = append(ito, v)
			}
			if lastPage {
				return false
			}
			return true
		})
	if err != nil {
		return ito,err
	}
	return ito, nil
}
