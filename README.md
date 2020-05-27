## ec2matcher

A commandline tool for finding ec2 instance types, the zones they are available in, and the on demand price

## Build
	go build -o ec2matcher main.go


## Usage
EC2 Instance Type matcher

Usage:
  ec2matcher [command]

Available Commands:
  help        Help about any command
  prices      Shows OnDemand pricing for a list of instances
  types       Matches ec2 instance types for desired specs
  zones       Shows instances in a list available in listed zones

Flags:
  -h, --help   help for ec2matcher

Use "ec2matcher [command] --help" for more information about a command.


## Examples

### Get types with 32 vcpus and 256GB of ram
	./ec2matcher types -c 32 -r 256
	Name: r5dn.8xlarge
	  Ram: 262144
	  Vcpus: 32
	Name: r5.8xlarge
	  Ram: 262144
	  Vcpus: 32
	Name: r5d.8xlarge
	  Ram: 262144
	  Vcpus: 32
	Name: r5n.8xlarge
	  Ram: 262144
	  Vcpus: 32
	Name: r5a.8xlarge
	  Ram: 262144
	  Vcpus: 32

### Get types with similar ram and cpu to a specific type  
	./ec2matcher types -t c5.12xlarge
	Name: c5d.12xlarge
	  Ram: 98304
	  Vcpus: 48
	Name: c5.12xlarge
	  Ram: 98304
	  Vcpus: 48


### Get types' availability in a list of zones
	./ec2matcher zones m4.large,m5dn.large,m5ad.large us-east-1a,us-east-1b,us-east-1c,us-east-1e,us-east-1f
	Instancs available in all zones:
	m4.large
	Instances with limited availability:
	m5dn.large: [us-east-1a us-east-1f us-east-1c us-east-1b]
	m5ad.large: [us-east-1a us-east-1c us-east-1f us-east-1b]

### Get prices for a list of types
	./ec2matcher prices m5dn.large,m5ad.large,m5d.large,t3.large,t2.large,m4.large,m5n.large,m6g.large,t3a.large,m5.large,m5a.large
	t3a.large: 0.0752000000
	m6g.large: 0.0770000000
	m5ad.large: 0.1030000000
	m5d.large: 0.1130000000
	t3.large: 0.0832000000
	t2.large: 0.0928000000
	m4.large: 0.1000000000
	m5n.large: 0.1190000000
	m5.large: 0.0960000000
	m5dn.large: 0.1360000000
	m5a.large: 0.0860000000

