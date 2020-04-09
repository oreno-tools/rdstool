package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	_ "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/kyokomi/emoji"
	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	AppVersion = "0.0.4"
)

var (
	argVersion    = flag.Bool("version", false, "バージョンを出力.")
	argModify     = flag.Bool("modify", false, "パラメータの更新.")
	argFailover   = flag.Bool("failover", false, "DB インスタンスのフェイルオーバーを開始.")
	argInstances  = flag.Bool("instances", false, "クラスタの DB インスタンス一覧を取得.")
	argInstance   = flag.String("instance", "", "DB インスタンス名を指定.")
	argClass      = flag.String("class", "", "DB インスタンスクラスを指定.")
	argCluster    = flag.String("cluster", "", "Aurora クラスタ名を指定.")
	argRestart    = flag.Bool("restart", false, "DB インスタンスの再起動を実施.")
	argParamGroup = flag.String("param-group", "", "DB パラメータグループの名前を指定")
	argParamName  = flag.String("param-name", "", "パラメータの名前を指定")
	argRatio      = flag.Float64("ratio", 0, "指定可能なパラメータ値のメモリに対する割合を指定.")

	svc   = rds.New(session.New())
	mega  = emoji.Sprint(":mega:")
	sushi = emoji.Sprint(":sushi:")
	warn  = emoji.Sprint(":beer:")
)

func main() {
	flag.Parse()

	if *argVersion {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	var clusterName string
	if *argCluster == "" {
		if os.Getenv("CLUSTER_NAME") != "" {
			clusterName = os.Getenv("CLUSTER_NAME")
		} else {
			fmt.Printf("%s`-cluster` パラメータ又は環境変数 `CLUSTER_NAME` を指定して下さい.\n", warn)
			os.Exit(1)
		}
	} else if *argCluster != "" {
		clusterName = *argCluster
	} else {
		fmt.Printf("%s`-cluster` パラメータ又は環境変数 `CLUSTER_NAME` を指定して下さい.\n", warn)
		os.Exit(1)
	}

	var paramGroup string
	if *argParamGroup == "" {
		if os.Getenv("PARAMETER_NAME") != "" {
			paramGroup = os.Getenv("PARAMETER_NAME")
		} else {
			fmt.Printf("%s`-param-group` パラメータ又は環境変数 `PARAMETER_NAME` を指定して下さい.\n", warn)
			os.Exit(1)
		}
	} else if *argParamGroup != "" {
		paramGroup = *argParamGroup
	} else {
		fmt.Printf("%s`-param-group` パラメータ又は環境変数 `CLUSTER_NAME` を指定して下さい.\n", warn)
		os.Exit(1)
	}

	fmt.Printf("%scluster: \x1b[31m%s\x1b[0m\tparameter group: \x1b[31m%s\x1b[0m\n", sushi, clusterName, paramGroup)

	dbInstances := getClusterInstances(clusterName)

	if dbInstances == nil {
		fmt.Printf("%sDB インスタンスが存在していません.\n", warn)
		os.Exit(1)
	}

	if *argInstances {
		printTable(dbInstances, "instance")
		os.Exit(0)
	}

	if !*argModify && *argParamName != "" {
		params := printParams(paramGroup, *argParamName)
		printTable(params, "param")
		os.Exit(0)
	}

	if *argModify && *argClass != "" {
		targetDBInstance := selectModifyTarget(dbInstances)
		fmt.Printf("%s DB インスタンス \x1b[31m%s\x1b[0m のインスタンスクラスを変更します. インスタンスクラスは \x1b[31m%s\x1b[0m です.\n", mega, targetDBInstance, *argClass)
		fmt.Printf("処理を継続しますか? (y/n): ")
		var stdin string
		fmt.Scan(&stdin)
		switch stdin {
		case "y", "Y":
			dbInstanceModifyStatus := executeInstanceClassModify(targetDBInstance, *argClass)
			if dbInstanceModifyStatus == "" {
				fmt.Printf("DB インスタンスのクラス変更に失敗しました.")
				os.Exit(1)
			}
			fmt.Printf("DB インスタンスのクラス変更中")
			// 泣きの wait
			for i := 0; i < 10; i++ {
				fmt.Printf(".")
				time.Sleep(time.Second * 1)
			}
			for {
				st, _, _ := getInstanceStatus(targetDBInstance)
				if st == "available" {
					fmt.Printf("\nDB インスタンスクラス変更完了.\n")
					os.Exit(0)
				}
				fmt.Printf(".")
				time.Sleep(time.Second * 5)
			}
		case "n", "N":
			fmt.Println("処理を停止します.")
			os.Exit(0)
		default:
			fmt.Println("処理を停止します.")
			os.Exit(0)
		}
		// printTable(dbInstances, "instance")
	}

	if *argFailover {
		targetDBInstance := selectFailoverTarget(dbInstances)
		fmt.Printf("%s DB クラスタ \x1b[31m%s\x1b[0m をフェイルーバーします. フェイルオーバー先は \x1b[31m%s\x1b[0m です.\n", mega, clusterName, targetDBInstance)
		fmt.Printf("処理を継続しますか? (y/n): ")
		var stdin string
		fmt.Scan(&stdin)
		switch stdin {
		case "y", "Y":
			dbClusterFailoverStatus := executeClusterFailover(clusterName, targetDBInstance)
			if dbClusterFailoverStatus == "" {
				fmt.Printf("DB クラスタのフェイルーバーに失敗しました.")
				os.Exit(1)
			}
			fmt.Printf("DB クラスタのフェイルオーバー実行中")
			for {
				st, _, _ := getInstanceStatus(targetDBInstance)
				dbInstances := getClusterInstances(clusterName)
				w := getWriteInstance(dbInstances)
				if st == "available" && w == targetDBInstance {
					fmt.Printf("\nDB クラスタフェイルオーバー完了.\n")
					os.Exit(0)
				}
				fmt.Printf(".")
				time.Sleep(time.Second * 5)
			}
		case "n", "N":
			fmt.Println("処理を停止します.")
			os.Exit(0)
		default:
			fmt.Println("処理を停止します.")
			os.Exit(0)
		}
	}

	if *argRestart {
		var restartDBInstanceName string
		if *argInstance == "" {
			restartDBInstanceName = selectRestartTarget(dbInstances)
		} else {
			restartDBInstanceName = *argInstance
		}
		fmt.Printf("%s DB インタンス \x1b[31m%s\x1b[0m を再起動します.\n", mega, restartDBInstanceName)
		fmt.Printf("処理を継続しますか? (y/n): ")
		var stdin string
		fmt.Scan(&stdin)
		switch stdin {
		case "y", "Y":
			dbInstanceStatus := restartDBInstance(restartDBInstanceName, *argFailover)
			if dbInstanceStatus == "" {
				fmt.Printf("DB インスタンスの再起動に失敗しました.")
				os.Exit(1)
			}
			fmt.Printf("DB インスタンスを再起動中")
			for {
				st, _, _ := getInstanceStatus(restartDBInstanceName)
				if st == "available" {
					fmt.Printf("\nDB インスタンス再起動完了.\n")
					os.Exit(0)
				}
				fmt.Printf(".")
				time.Sleep(time.Second * 5)
			}
		case "n", "N":
			fmt.Println("処理を停止します.")
			os.Exit(0)
		default:
			fmt.Println("処理を停止します.")
			os.Exit(0)
		}
	}

	if *argModify && *argParamName != "" {
		var latest_value string
		if *argRatio != 0 {
			latest_value = genParameterValue(*argRatio)
		} else {
			fmt.Println("`-rasio` パラメータを指定して下さい.")
			os.Exit(1)
		}

		params := printParams(paramGroup, *argParamName)
		if len(params) != 1 {
			fmt.Println("DB パラメータの指定に誤りがあります. パラメータ名を確認して下さい.")
			os.Exit(1)
		}
		fmt.Printf("%s DB パラメータ \x1b[31m%s\x1b[0m の値を \x1b[31m%s\x1b[0m に変更します.\n", mega, *argParamName, latest_value)
		fmt.Printf("処理を継続しますか? (y/n): ")
		var stdin string
		fmt.Scan(&stdin)
		switch stdin {
		case "y", "Y":
			dbInstance := getWriteInstance(dbInstances)
			fmt.Println("DB パラメータを更新します.")
			modifyValue(paramGroup, *argParamName, latest_value)
			fmt.Printf("DB パラメータ更新中")
			for {
				if getParameterStatus(dbInstance, paramGroup) == "pending-reboot" {
					fmt.Printf("\n%s DB パラメータ更新完了. DB インスタンスの再起動が必要です. `-restart -instance=xxxxxxxx` オプションを指定して DB インスタンスを再起動して下さい.\n", mega)
					break
				} else if getParameterStatus(dbInstance, paramGroup) == "" {
					fmt.Println("DB パラメータの更新に失敗しました.")
					os.Exit(1)
				}
				fmt.Printf(".")
				time.Sleep(time.Second * 5)
			}
			os.Exit(0)
		case "n", "N":
			fmt.Println("処理を停止します.")
			os.Exit(0)
		default:
			fmt.Println("処理を停止します.")
			os.Exit(0)
		}
	}
	flag.PrintDefaults()
}

func printTable(data [][]string, t string) {
	table := tablewriter.NewWriter(os.Stdout)
	if t == "instance" {
		// table.SetHeader([]string{"InstanceIdentifier", "InstanceStatus", "Writer", "ParameterApplyStatus", "ClusterParameterGroupStatus", "PromotionTier"})
		table.SetHeader([]string{"InstanceIdentifier", "InstanceStatus", "Writer", "InstanceClass", "ParameterApplyStatus", "ClusterParameterGroupStatus"})
		for _, value := range data {
			if value[2] == "true" {
				for i, e := range value {
					value[i] = fmt.Sprintf("\x1b[31m%s\x1b[0m", e)
				}
			}
			table.Append(value)
		}
	} else if t == "param" {
		table.SetHeader([]string{"ParameterName", "ParameterValue", "DataType", "AllowedValues"})
		table.AppendBulk(data)
	}

	table.Render()
}

func genParameterValue(value float64) string {
	v := 8192.0 / value
	parameter := fmt.Sprintf("{DBInstanceClassMemory/%s}", fmt.Sprint(int(v)))

	return parameter
}

func getParameterStatus(dbInstance string, paramGroup string) string {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstance),
	}

	result, err := svc.DescribeDBInstances(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return ""
	}

	var st string
	for _, r := range result.DBInstances {
		for _, p := range r.DBParameterGroups {
			if *p.DBParameterGroupName == paramGroup {
				st = *p.ParameterApplyStatus
			}
		}
	}

	return st
}

// Restart DB Instance
func restartDBInstance(dbInstance string, failover bool) string {
	input := &rds.RebootDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbInstance),
		// Aurora クラスタではエラーになるので, 何らかの回避方法で RDS と Aurora 両方に対応出来るようにする... いつか
		// ForceFailover:    aws.Bool(failover),
	}

	result, err := svc.RebootDBInstance(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return ""
	}

	st := *result.DBInstance.DBInstanceStatus
	return st
}

// Modify DB Instance class
func executeInstanceClassModify(instanceName string, instanceClass string) string {
	input := &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(instanceName),
		DBInstanceClass:      aws.String(instanceClass),
		ApplyImmediately:     aws.Bool(true),
	}

	result, err := svc.ModifyDBInstance(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return ""
	}

	st := *result.DBInstance.DBInstanceStatus
	return st
}

// Failover DB Cluster
func executeClusterFailover(clusterName string, targetDBInstance string) string {
	input := &rds.FailoverDBClusterInput{
		DBClusterIdentifier:        aws.String(clusterName),
		TargetDBInstanceIdentifier: aws.String(targetDBInstance),
	}

	result, err := svc.FailoverDBCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return ""
	}

	st := *result.DBCluster.Status
	return st
}

func getClusterInstances(clusterName string) [][]string {
	input := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterName),
	}

	result, err := svc.DescribeDBClusters(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}

	// fmt.Println(result.DBClusters[0].DBClusterMembers)
	var instances [][]string
	for _, i := range result.DBClusters[0].DBClusterMembers {
		// tier := strconv.FormatInt(*i.PromotionTier, 10)
		st, cl, ps := getInstanceStatus(*i.DBInstanceIdentifier)
		instance := []string{
			*i.DBInstanceIdentifier,
			st,
			strconv.FormatBool(*i.IsClusterWriter),
			cl,
			ps,
			*i.DBClusterParameterGroupStatus,
			// tier,
		}
		instances = append(instances, instance)
	}
	return instances
}

func getWriteInstance(dbInstances [][]string) string {
	var writer string
	for _, i := range dbInstances {
		if i[2] == "true" {
			writer = i[0]
		}
	}
	return writer
}

func selectModifyTarget(dbInstances [][]string) string {
	var targets []string
	for _, i := range dbInstances {
		targets = append(targets, i[0])
	}
	prompt := promptui.Select{
		Label: "変更する DB インスタンスを選択して下さい.",
		Items: targets,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}

	return result
}

func selectFailoverTarget(dbInstances [][]string) string {
	var targets []string
	for _, i := range dbInstances {
		if i[2] != "true" {
			targets = append(targets, i[0])
		}
	}
	prompt := promptui.Select{
		Label: "フェイルオーバー先の DB インスタンスを選択して下さい.",
		Items: targets,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}

	return result
}

func selectRestartTarget(dbInstances [][]string) string {
	var targets []string
	for _, i := range dbInstances {
		targets = append(targets, i[0])
	}
	prompt := promptui.Select{
		Label: "再起動する DB インスタンスを選択して下さい.",
		Items: targets,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}

	return result
}

func getInstanceStatus(dbInstance string) (string, string, string) {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstance),
	}

	result, err := svc.DescribeDBInstances(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return "", "", ""
	}

	st := *result.DBInstances[0].DBInstanceStatus
	cl := *result.DBInstances[0].DBInstanceClass
	ps := *result.DBInstances[0].DBParameterGroups[0].ParameterApplyStatus
	return st, cl, ps
}

func modifyValue(paramGroup string, paramName string, paramValue string) {
	input := &rds.ModifyDBParameterGroupInput{
		DBParameterGroupName: aws.String(paramGroup),
		Parameters: []*rds.Parameter{
			{
				ApplyMethod:    aws.String("pending-reboot"),
				ParameterName:  aws.String(paramName),
				ParameterValue: aws.String(paramValue),
			},
		},
	}

	_, err := svc.ModifyDBParameterGroup(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return
	}
}

func printParams(paramGroup string, paramNamePrefix string) [][]string {
	input := &rds.DescribeDBParametersInput{
		DBParameterGroupName: aws.String(paramGroup),
	}

	var params [][]string
	for {
		result, err := svc.DescribeDBParameters(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				fmt.Println(aerr.Error())
			} else {
				fmt.Println(err.Error())
			}
			return nil
		}
		for _, p := range result.Parameters {
			if paramNamePrefix != "" {
				if strings.Contains(*p.ParameterName, paramNamePrefix) {
					var pv string
					if p.ParameterValue == nil {
						pv = "N/A"
					} else {
						pv = *p.ParameterValue
					}
					var avs string
					if p.AllowedValues == nil {
						avs = "N/A"
					} else {
						avs = *p.AllowedValues
					}
					param := []string{*p.ParameterName, pv, *p.DataType, avs}
					params = append(params, param)
				}
			} else {
				// Bug
				param := []string{*p.ParameterName, *p.ParameterValue, *p.DataType, *p.AllowedValues}
				params = append(params, param)
			}
		}
		if result.Marker == nil {
			break
		}
		input.SetMarker(*result.Marker)
		continue
	}

	return params
}
