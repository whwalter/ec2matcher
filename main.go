package main

import (
	"fmt"
	"sort"
	"log"
	"errors"
	"strings"
	"strconv"
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/spf13/cobra"
)

var awsSession *session.Session
var ec2Client *ec2.EC2

func main() {
	if err := newRootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ec2matcher",
		Short: "EC2 Instance Type sourcer",
	}
	cobra.OnInitialize(initConfig)
	cmd.AddCommand(newEC2TypeMatcherCommand())
	cmd.AddCommand(newEC2ZoneMatcherCommand())
	cmd.AddCommand(newEC2PriceMatcherCommand())
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
}


func newEC2TypeMatcherCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "types",
		Short: "Matches ec2 instance types for desired specs",
		RunE: matchEC2TypesPrintRunE,
	}

	cmd.PersistentFlags().StringP("type", "t", "", "Source type to match against")
	cmd.PersistentFlags().Int64P("ram", "r", 0, "Ram in GB to match against")
	cmd.PersistentFlags().Int64P("vcpu", "c", 0, "vcpu count to match against")

	return cmd
}

// TODO: this should support flags for baremetal, boosted, and gpu types for future filtering
func matchEC2TypesPrintRunE(cmd *cobra.Command, args []string) error {

	matchType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}	

	ram, err := cmd.Flags().GetInt64("ram")
	if err != nil {
		return err
	}	

	vcpu, err := cmd.Flags().GetInt64("vcpu")
	if err != nil {
		return err
	}	

	matches,err := matchEC2Types(matchType, ram, vcpu)
	if err != nil {
		return err
	}

	// []*ec2InstanceTypeInfo
	for _,v := range matches {
		fmt.Printf("Name: %s\n  Ram: %d\n  Vcpus: %d\n", *v.InstanceType, *v.MemoryInfo.SizeInMiB, *v.VCpuInfo.DefaultVCpus)
	}
	return nil
}

func matchEC2Types(matchType string, ram, vcpu int64) ([]*ec2.InstanceTypeInfo,error) {

	// TODO: split this out and support other match types: Ram, CPU Count
	if matchType == "" && ram == 0 && vcpu == 0 {
		return []*ec2.InstanceTypeInfo{}, errors.New("At least one of type, ram, or vcpu must be specified")
	}
	var mem, cpu *string
	if matchType != "" {
		typeInput := ec2.DescribeInstanceTypesInput{}
		mt := aws.String(matchType)
		typeInput.SetInstanceTypes([]*string{mt})
		err := typeInput.Validate()
		if err != nil {
			return []*ec2.InstanceTypeInfo{}, err
		}
		// err filters for invalid instance type strings
		dit, err := ec2Client.DescribeInstanceTypes(&typeInput)
		if err != nil {
			return []*ec2.InstanceTypeInfo{}, err
		}

		// Only one instance type is returned for explicit name match
		primaryType := dit.InstanceTypes[0]
		mem = aws.String(fmt.Sprint(*primaryType.MemoryInfo.SizeInMiB))
		cpu = aws.String(fmt.Sprint(*primaryType.VCpuInfo.DefaultVCpus))
	}

	if mem == nil && ram != 0 {
		mem = aws.String(strconv.FormatInt(ram * 1024, 10))
	}
	if cpu == nil && vcpu != 0 {
		cpu = aws.String(strconv.FormatInt(vcpu, 10))
	}

	matchers := []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("bare-metal"),
				Values: []*string{
					aws.String("false"),
					},
		  	 }}

	if mem != nil {
		matchers = append(matchers, &ec2.Filter{
				Name: aws.String("memory-info.size-in-mib"),
				Values: []*string{ mem },
				})
	}

	if cpu != nil {
		matchers = append(matchers, &ec2.Filter{
				Name: aws.String("vcpu-info.default-vcpus"),
				Values: []*string{ cpu },
				})

	}

	typeInput := ec2.DescribeInstanceTypesInput{
		DryRun: aws.Bool(false),
		Filters: matchers,
		MaxResults: aws.Int64(5),
	}
	err := typeInput.Validate()
	if err != nil {
		return []*ec2.InstanceTypeInfo{}, err
	}

	// []*ec2InstanceTypeInfo
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
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return errors.New("Comma delimited strings for desired ec2 types and zones are required")
	}
	types := strings.Split(args[0], ",")
	zones := strings.Split(args[1], ",")
	matches, err := matchEC2Zones(types, zones)
	if err != nil {
		return err
	}	
	for k := range matches{
		fmt.Println(k)
	}
	return nil
}
func matchEC2Zones(types []string, zones []string) (map[string]*instanceType, error) {
	validTypes := map[string]*instanceType{}

	awsTypes := aws.StringSlice(types)
	awsZones := aws.StringSlice(zones)

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

	offeringInput := ec2.DescribeInstanceTypeOfferingsInput{
				Filters: offeringFilters,
				LocationType: aws.String("availability-zone"),
				MaxResults: aws.Int64(5),
			}
	err := offeringInput.Validate()
	if err != nil {
		return validTypes, err
	}

	typesInZones, err := filterInstanceTypeOfferings(offeringInput, ec2Client)
	if err != nil {
		return validTypes, err
	}

	// Populate map of valid instanceTypes
	for _, v := range typesInZones {
		// if this is the first match, add an instanceType to the map, else append the location to the list of locations
		if validTypes[aws.StringValue(v.InstanceType)] == nil {
			validTypes[aws.StringValue(v.InstanceType)] = &instanceType{
									Name: aws.StringValue(v.InstanceType),
									Locations: []string{aws.StringValue(v.Location)},
									}
		} else {
		validTypes[aws.StringValue(v.InstanceType)].Locations = append(validTypes[aws.StringValue(v.InstanceType)].Locations, aws.StringValue(v.Location))
		}
	}

	// If a type isn't available in all desired zones, remove it from the list of options
	for k,v := range validTypes {
		if !compareUnsortedStringSlice(zones, v.Locations){
			delete(validTypes, k)
		}
	}
	return validTypes, nil
}

func newEC2PriceMatcherCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "prices",
		Short: "Shows OnDemand pricing for a list of instances",
		Args: cobra.ExactValidArgs(1),
		RunE: matchEC2PricesPrintRunE,
	}

	return cmd
}


func matchEC2PricesPrintRunE(cmd *cobra.Command, args []string) error {
	types := strings.Split(args[0], ",")
	prices, err :=  reportPricing(types)
	if err != nil {
		return err
	}

	for k,v := range prices {
		fmt.Printf("%s: %s\n", k, v)
	}
	return nil
}
func reportPricing(types []string) ( map[string]string, error) {

	awsSession = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	pc := pricing.New(awsSession)

	ft := aws.String("TERM_MATCH")

	defaultFilters := []*pricing.Filter{
		&pricing.Filter{
			Field: aws.String("ServiceCode"),
			Type: ft,
			Value: aws.String("AmazonEC2"),
		},
		&pricing.Filter{
			Field: aws.String("operatingsystem"),
			Type: ft,
			Value: aws.String("Linux"),
		},
		&pricing.Filter{
			Field: aws.String("tenancy"),
			Type: ft,
			Value: aws.String("shared"),
		},
		&pricing.Filter{
			Field: aws.String("preInstalledSw"),
			Type: ft,
			Value: aws.String("NA"),
		},
		&pricing.Filter{
			Field: aws.String("capacitystatus"),
			Type: ft,
			Value: aws.String("UnusedCapacityReservation"),
		},
		&pricing.Filter{
			Field: aws.String("location"),
			Type: ft,
			Value: aws.String("US East (N. Virginia)"),
		},
	}

	prices := map[string]string{}
	for _, t := range types {
		filters := []*pricing.Filter{
				&pricing.Filter{
					Field: aws.String("instancetype"),
					Type: ft,
					Value: aws.String(t),
				},
			}
		gpi := pricing.GetProductsInput{
			Filters: append(filters, defaultFilters...),
			FormatVersion: aws.String("aws_v1"),
			ServiceCode: aws.String("AmazonEC2"),
			}
		err := gpi.Validate()
		if err != nil {
			return map[string]string{}, err
		}

		values, err := filterPrices(gpi, pc)
		if err != nil {
			return map[string]string{}, err
		}
		
		for _, v := range values {
			it, p, e := parsePrice(v)
			if e != nil {
				return map[string]string{}, e
			}
			prices[*it] = *p
		}
	}
	return prices, nil
}


func parsePrice(i map[string]interface{}) (it *string, price *string, e error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil,nil, err
	}

	pr := ProductSpec{}
	err = json.Unmarshal(b, &pr)
	if err != nil {
		return nil,nil, err
	}

	var p string
	for _, z := range pr.Terms.OnDemand {
		for _, v := range z.PriceDimensions {
			p = v.PricePerUnit["USD"]
		}
	}
	return &pr.Product.Attributes.InstanceType, &p, nil
}

func filterPrices(describer pricing.GetProductsInput, client *pricing.Pricing) (pricingList []aws.JSONValue, e error) {
	var pl []aws.JSONValue
	err := client.GetProductsPages(
		&describer,
		func(page *pricing.GetProductsOutput, lastPage bool) bool{
			for _,v := range page.PriceList {
				pl = append(pl, v)
			}
			if lastPage {
				return false
			}
			return true
		})
	if err != nil {
		return pl,err
	}
	return pl, nil
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

// Terms - Generated from json map[string]interface{}
type Terms struct {
	OnDemand map[string]InstanceData
}

// InstanceData - Generated from json map[string]interface{}
type InstanceData struct {
	EffectiveDate string
	OfferTermCode string
	PriceDimensions map[string]PriceDimensions
	Sku string
	*TermAttributes
}

// TermAttributes - Generated from json map[string]interface{}, nil in sample data
type TermAttributes struct {
}

// PriceDimensions - Generated from json map[string]interface{}
type PriceDimensions struct {
	*PriceData `json:",string"`
}

// PriceData - Generated from json map[string]interface{}
type PriceData struct {
	AppliesTo []string
	BeginRange string
	Description string
	EndRange string
	PricePerUnit map[string]string
	RateCode string
	Unit string
}

// PricePerUnit - Generated from json map[string]interface{}
type PricePerUnit struct {
	USD string
}

// Attributes - Generated from json map[string]interface{}
type Attributes struct {
	EnhancedNetworkingSupported string
	OperatingSystem string
	InstanceFamily string
	IntelAvxAvailable string
	NetworkPerformance string
	Tenancy string
	ClockSpeed string
	LicenseModel string
	PhysicalProcessor string
	Capacitystatus string
	DedicatedEbsThroughput string
	Location string
	ProcessorFeatures string
	Servicename string
	CurrentGeneration string
	Ecu string
	ProcessorArchitecture string
	Servicecode string
	Usagetype string
	Vcpu string
	IntelAvx2Available string
	Memory string
	PreInstalledSw string
	Storage string
	InstanceType string
	LocationType string
	NormalizationSizeFactor string
	Operation string
	Instancesku string
	IntelTurboAvailable string
}

// Product - Generated from json map[string]interface{}
type Product struct {
	ProductFamily string
	Sku string
	Attributes Attributes `json:"attributes"`
}

// ProductSpec - Generated from json map[string]interface{}
type ProductSpec struct {
	ServiceCode string
	Terms Terms
	Version string
	Product Product
	PublicationDate string
}
