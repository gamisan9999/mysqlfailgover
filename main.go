package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/codegangsta/cli"
)

// 指定したInstance IDが登録されているRouteTable IDを返す
// ToDo: MySQL VIP用RouteTableが複数返されたらどーすんべ対応
func instanceIDToRouteTableID(svc *ec2.EC2, InstanceID string) string {
	findOpts := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("route.instance-id"),
				Values: []*string{
					aws.String(InstanceID),
				},
			},
		},
	}
	resp, err := svc.DescribeRouteTables(findOpts)
	var routetableID string
	for _, idx := range resp.RouteTables {
		for _, rtbs := range idx.Associations {
			routetableID = *rtbs.RouteTableId
		}
	}
	if err != nil {
		panic(err)
	}
	return routetableID
}

// 指定したRouteTableIdに登録されているDestination InstanceIDを変更する
func replaceRouteTable(svc *ec2.EC2, mysqlVip string, routeTableID string, instanceID string) *ec2.ReplaceRouteOutput {
	params := &ec2.ReplaceRouteInput{
		DestinationCidrBlock: aws.String(mysqlVip),
		RouteTableId:         aws.String(routeTableID),
		DryRun:               aws.Bool(false),
		InstanceId:           aws.String(instanceID),
	}
	resp, err := svc.ReplaceRoute(params)
	if err != nil {
		panic(err)
	}
	return resp
}

// 指定したEC2 Private IPからinstance idを返す
func ipToInstanceID(svc *ec2.EC2, privateIpaddress string) string {
	findOpts := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("private-ip-address"),
				Values: []*string{
					aws.String(privateIpaddress),
				},
			},
		},
	}
	resp, err := svc.DescribeInstances(findOpts)
	if err != nil {
		panic(err)
	}
	var instanceID string
	for _, idx := range resp.Reservations {
		for _, inst := range idx.Instances {
			instanceID = *inst.InstanceId
		}
	}
	return instanceID
}

func main() {
	app := cli.NewApp()
	app.Name = "mysqlfailgover"
	app.Usage = "set MHA failover script. MySQL Master EC2 instance routetable change"
	app.Version = "0.0.1"
	// MHA failover scriptの引数よりRouteTableを書き換えるための必要な引数をセットする
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "mysql_master_vip",
			Value: "mysql master host vip/CIDR",
			Usage: "mha executable failover command arguments --mysql_master_vip, ex) --mysql_master_vip=192.168.10.1/32",
		},
		cli.StringFlag{
			Name:  "command",
			Value: "start, stop_ssh, status",
			Usage: "mha executable failover command arguments --command",
		},
		cli.StringFlag{
			Name:  "orig_master_host, host",
			Value: "original master host",
			Usage: "mha executable failover command arguments --orig_master_host",
		},
		cli.StringFlag{
			Name:  "orig_master_ip, ip",
			Value: "original master host ip",
			Usage: "mha executable failover command arguments --orig_master_ip",
		},
		cli.StringFlag{
			Name:  "new_master_host",
			Value: "new master host",
			Usage: "mha executable failover command arguments --new_master_host",
		},
		cli.StringFlag{
			Name:  "new_master_ip",
			Value: "new master host ip",
			Usage: "mha executable failover command arguments --new_master_ip",
		},
		cli.StringFlag{
			Name:  "orig_master_port",
			Value: "no use",
		},
		cli.StringFlag{
			Name:  "ssh_user",
			Value: "ssh_username",
			Usage: "no use",
		},
		cli.StringFlag{
			Name:  "new_master_port",
			Value: "new master port",
			Usage: "no use",
		},
		cli.StringFlag{
			Name:  "orig_master_user",
			Value: "orig_master_user(mha)",
			Usage: "no use",
		},
		cli.StringFlag{
			Name:  "orig_master_password",
			Value: "orig_master_password",
			Usage: "no use",
		},
		cli.StringFlag{
			Name:  "new_master_user",
			Value: "new_master_user(mha)",
			Usage: "no use",
		},
		cli.StringFlag{
			Name:  "new_master_password",
			Value: "new_master_password(mha user's password)",
			Usage: "no use",
		},
		cli.StringFlag{
			Name:  "new_master_ssh_user",
			Value: "new_master_ssh_user",
			Usage: "no use",
		},
		cli.StringFlag{
			Name:  "orig_master_ssh_user",
			Value: "orig_master_ssh_user",
			Usage: "no use",
		},
	}

	app.Action = func(c *cli.Context) {
		fmt.Println(c.NumFlags())
		if c.NumFlags() == 0 {
			fmt.Println(app.Name, " --help to view usage.")
			os.Exit(1)
		}
		svc := ec2.New(session.New(), &aws.Config{Region: aws.String("ap-northeast-1")})
		if c.String("command") == "status" {
			origMasterInstanceID := ipToInstanceID(svc, c.String("orig_master_ip"))
			routetableID := instanceIDToRouteTableID(svc, origMasterInstanceID)
			fmt.Println("orig_master_ip registration route table id: ", routetableID)
		}
		if c.String("command") == "start" {
			fmt.Println("mysql master vip:", c.String("mysql_master_vip"))
			fmt.Println("master_host:", c.String("orig_master_host"))
			fmt.Println("master_host_ip:", c.String("orig_master_host_ip"))
			// MHA failoverscriptの引数に渡されるprivate ip addressからInstance IDを取得する
			origMasterInstanceID := ipToInstanceID(svc, c.String("orig_master_ip"))
			newMasterInstanceID := ipToInstanceID(svc, c.String("new_master_ip"))
			fmt.Println(c.String("orig_master_ip"), "=", origMasterInstanceID)
			fmt.Println(c.String("new_master_ip"), "=", newMasterInstanceID)
			routetableID := instanceIDToRouteTableID(svc, origMasterInstanceID)
			fmt.Println("orig_master_ip registration route table id: ", routetableID)
			fmt.Println("route table id", routetableID, ",replace destination instance ", origMasterInstanceID, "to", newMasterInstanceID)
			replaceRouteTable(svc, c.String("mysql_master_vip"), routetableID, newMasterInstanceID)
			fmt.Println("route table,", routetableID, "replaced.")
		}
	}
	app.Run(os.Args)
}
