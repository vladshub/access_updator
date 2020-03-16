package main

import (
	"flag"
	"net"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sirupsen/logrus"
)

var (
	user      string
	group     string
	ipAddress string
)

func init() {
	flag.StringVar(&user, "user", "", "the name of the user we want to give access to")
	flag.StringVar(&group, "group", "", "the AWS group id")
	flag.StringVar(&ipAddress, "ip", "", "the IP address we want to add")
}

func main() {
	flag.Parse()
	usage("user", user)
	usage("group", group)
	usage("ip", ipAddress)

	ip := net.ParseIP(ipAddress)
	if ip.IsInterfaceLocalMulticast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLoopback() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		logrus.Fatal("invalid ip address ", ipAddress)
	}

	ipAddress += "/32"

	sess, err := session.NewSession()
	if err != nil {
		logrus.WithError(err).Fatal("failed creating session with AWS")
	}
	svc := ec2.New(sess)

	qinput := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Purpose"),
				Values: []*string{
					aws.String("test"),
				},
			},
		},
	}

	qresult, err := svc.DescribeSecurityGroups(qinput)
	if handleError(err, "failed querying for security group") {
		return
	}

	g := qresult.SecurityGroups[0]
	if g == nil {
		logrus.Fatal("no groups found")
		return
	}

	if aws.StringValue(g.GroupId) != group {
		logrus.Fatalf("invalid group found. Expected %s got %s", group, g.GroupId)
	}

	for _, perm := range g.IpPermissions {
		for _, r := range perm.IpRanges {
			if aws.StringValue(r.Description) == user && aws.StringValue(r.CidrIp) == ipAddress {
				logrus.Info("Ip already assigned correctly")
				os.Exit(0)
			} else if aws.StringValue(r.Description) == user && aws.StringValue(r.CidrIp) != ipAddress {
				d := &ec2.RevokeSecurityGroupIngressInput{
					GroupId: g.GroupId,
					IpPermissions: []*ec2.IpPermission{
						{
							FromPort:   aws.Int64(443),
							IpProtocol: aws.String("https"),
							IpRanges: []*ec2.IpRange{
								{
									CidrIp:      aws.String(ipAddress),
									Description: aws.String(user),
								},
							},
							ToPort: aws.Int64(443),
						},
					},
				}
				_, err := svc.RevokeSecurityGroupIngress(d)
				if handleError(err, "failed removing existing rule") {
					return
				}
				logrus.Info("successfully removed existing rule")

			}
		}
	}

	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(group),
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(443),
				IpProtocol: aws.String("https"),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      aws.String(ipAddress),
						Description: aws.String(user),
					},
				},
				ToPort: aws.Int64(443),
			},
		},
	}

	result, err := svc.AuthorizeSecurityGroupIngress(input)
	if handleError(err, "failed creating the rule") {
		return
	}
	logrus.Info(result)
}

func usage(name, val string) {
	if val == "" {
		logrus.Errorf("missing required argument %s", name)
		flag.Usage()
		os.Exit(1)
	}
}

func handleError(err error, message string) bool {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			logrus.WithError(aerr).Fatal(message)
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			logrus.WithError(err).Fatal(message)
		}
		return true
	}
	return false
}
