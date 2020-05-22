package main

import (
	"fmt"
	"sort"
	"log"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	//	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
)


// TODO: fill in instanceyTypeMatcher struct to support command line flags and settings
type instanceTypeMatcher struct {
	name string
	cpu   int64
	ram   int64
	metal bool
        boosted bool
	gpu bool        
}

// TODO: fill in regionMatcher struct to support command line flags and settings
type zoneMatcher struct {
	zones string	
}

// TODO: fill in priceMatcher struct to support command line flags and settings
type priceMatcher struct {
	price int64
	variance int64
}
var awsSession *session.Session
var ec2Client *ec2.EC2

var iType instanceTypeMatcher
var zones zoneMatcher
var prices priceMatcher
func main() {

	if err := newRootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ec2typer",
		Short: "EC2 Instance Type sourcer",
	}
	cobra.OnInitialize(initConfig)
	cmd.AddCommand(newEC2TypeMatcherCommand())
	cmd.AddCommand(newEC2ZoneMatcherCommand())
	return cmd
}

func initConfig() {
	if awsSession == nil {
		awsSession = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
	}

	if ec2Client == nil {
		ec2Client = ec2.New(awsSession)
	}
	zones = zoneMatcher{}
	prices = priceMatcher{}
}


func newEC2TypeMatcherCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "types",
		Short: "Matches ec2 instance types for desired specs",
		RunE: matchEC2TypesPrintRunE,
	}

	cmd.PersistentFlags().StringP("type", "t", "", "Source type to match against")
	cmd.PersistentFlags().Int64P("ram", "r", 0, "Ram to match against")
	cmd.PersistentFlags().Int64P("vcpu", "c", 0, "vcpu count to match against")

	return cmd
}

// TODO: this should support flags for baremetal, boosted, and gpu types for future filtering
func matchEC2TypesPrintRunE(cmd *cobra.Command, args []string) error {

	matchType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}	
/*
	ram, err := cmd.Flags().GetInt64("ram")
	if err != nil {
		return err
	}	
	cpu, err := cmd.Flags().GetInt64("vcpu")
	if err != nil {
		return err
	}	
*/
	matches,err := matchEC2Types(matchType)
	if err != nil {
		return err
	}
	for _,v := range matches {
		fmt.Printf("Name: %s\n  Ram: %d\n  Vcpus: %d\n", *v.InstanceType, *v.MemoryInfo.SizeInMiB, *v.VCpuInfo.DefaultVCpus)
	}
	return nil
}

func matchEC2Types(matchType string) ([]*ec2.InstanceTypeInfo,error) {

	//primaryType := aws.String("r5.8xlarge")
	primaryType := aws.String(matchType)
	typeInput := ec2.DescribeInstanceTypesInput{
		DryRun: aws.Bool(false),
		InstanceTypes: []*string{primaryType},
	}

	err := typeInput.Validate()
	if err != nil {
		return []*ec2.InstanceTypeInfo{}, err
	}
	dit, err := ec2Client.DescribeInstanceTypes(&typeInput)
	if err != nil {
		return []*ec2.InstanceTypeInfo{}, err
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
	typeInput = ec2.DescribeInstanceTypesInput{
		DryRun: aws.Bool(false),
		Filters: matchers,
		MaxResults: aws.Int64(5),
	}
	err = typeInput.Validate()
	if err != nil {
		return []*ec2.InstanceTypeInfo{}, err
	}
	its, err := filterInstanceTypes(typeInput, ec2Client)
	if err != nil {
		return []*ec2.InstanceTypeInfo{}, err
	}

	return its, nil
}

func newEC2ZoneMatcherCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "zones",
		Short: "Shows instances in a list available in listed zones",
		Args: cobra.ExactValidArgs(2),
		RunE: matchEC2ZonesPrintRunE,
	}

	return cmd
}

func matchEC2ZonesPrintRunE(cmd *cobra.Command, args []string) error {
//	types := strings.Split(args[0], ",")
	//zones := strings.Split(args[1], ",")
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return errors.New("Comma delimited strings for desired ec2 types and zones are required")
	}
	fmt.Printf("%v", args)
	matches, err := matchEC2Zones(args[0], args[1])
	if err != nil {
		return err
	}	
	for k := range matches{
		fmt.Println(k)
	}
	return nil
}
func matchEC2Zones(types string, zones string) (map[string]*instanceType, error) {
	validTypes := map[string]*instanceType{}

	awsTypes := aws.StringSlice(strings.Split(types, ","))
	awsZones := aws.StringSlice(strings.Split(zones, ","))

	offeringFilters := []*ec2.Filter{
				&ec2.Filter{
					Name: aws.String("location"),
					Values: awsZones,
				},
				&ec2.Filter{
					Name: aws.String("instance-type"),
					Values: awsTypes,
				},
			}
	fmt.Print(offeringFilters)
	offeringInput := ec2.DescribeInstanceTypeOfferingsInput{
				Filters: offeringFilters,
				LocationType: aws.String("availability-zone"),
				MaxResults: aws.Int64(5),
			}
	err := offeringInput.Validate()
	if err != nil {
		return validTypes, err
	}

	fmt.Printf("%v", offeringInput)
	typesInZones, err := filterInstanceTypeOfferings(offeringInput, ec2Client)
	if err != nil {
		return validTypes, err
	}
	fmt.Println(typesInZones)

	for _, v := range typesInZones {
		if validTypes[aws.StringValue(v.InstanceType)] == nil {
			validTypes[aws.StringValue(v.InstanceType)] = &instanceType{
									Name: aws.StringValue(v.InstanceType),
									Locations: []string{aws.StringValue(v.Location)},
									}
		} else {
		validTypes[aws.StringValue(v.InstanceType)].Locations = append(validTypes[aws.StringValue(v.InstanceType)].Locations, aws.StringValue(v.Location))
		}
	}

	for k,v := range validTypes {
		if !compareUnsortedStringSlice(strings.Split(zones, ","), v.Locations){
			delete(validTypes, k)
		}
	}
	return validTypes, nil
}


type instanceType struct {
	Name string
	Locations []string 
}
func filterInstanceTypes(describer ec2.DescribeInstanceTypesInput, client *ec2.EC2) (instanceTypes []*ec2.InstanceTypeInfo, e error) {
	its := []*ec2.InstanceTypeInfo{}
	err := client.DescribeInstanceTypesPages(
		&describer,
		func(page *ec2.DescribeInstanceTypesOutput, lastPage bool) bool{
			for _,v := range page.InstanceTypes {
				its = append(its,v)
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

func stringInSlice(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func compareUnsortedStringSlice(a,b []string) bool {
	if ( a == nil) != ( b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
