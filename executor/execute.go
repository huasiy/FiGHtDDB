package executor

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FiGHtDDB/parser"
	"github.com/FiGHtDDB/storage"
	_ "github.com/lib/pq"
)

type Db struct {
	dbname   string
	user     string
	password string
	port     int
	sslmode  string
}

func NewDb(dbname string, user string, password string, port int, sslmode string) *Db {
	db := new(Db)
	db.dbname = dbname
	db.user = user
	db.password = password
	db.port = port
	db.sslmode = sslmode

	return db
}

var db = NewDb("postgres", "postgres", "postgres", 5700, "disable")

var (
	ServerIp   = ""
	ServerName = ""
)

type Tuples struct {
	colNames []string
	resp     *[][]byte
}

var tableList1 = make([]string, 0)
var tableList2 = make([]string, 0)
var tableList3 = make([]string, 0)
var tableList4 = make([]string, 0)

var mutex sync.Mutex

func Strval(value interface{}) string {
	var key string
	if value == nil {
		return key
	}

	switch value.(type) {
	case float64:
		ft := value.(float64)
		key = strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		ft := value.(float32)
		key = strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		it := value.(int)
		key = strconv.Itoa(it)
	case uint:
		it := value.(uint)
		key = strconv.Itoa(int(it))
	case int8:
		it := value.(int8)
		key = strconv.Itoa(int(it))
	case uint8:
		it := value.(uint8)
		key = strconv.Itoa(int(it))
	case int16:
		it := value.(int16)
		key = strconv.Itoa(int(it))
	case uint16:
		it := value.(uint16)
		key = strconv.Itoa(int(it))
	case int32:
		it := value.(int32)
		key = strconv.Itoa(int(it))
	case uint32:
		it := value.(uint32)
		key = strconv.Itoa(int(it))
	case int64:
		it := value.(int64)
		key = strconv.FormatInt(it, 10)
	case uint64:
		it := value.(uint64)
		key = strconv.FormatUint(it, 10)
	case string:
		key = value.(string)
	case []byte:
		key = string(value.([]byte))
	default:
		newValue, _ := json.Marshal(value)
		key = string(newValue)
	}

	return key
}

// return type?
// consider we may project, union and join later
func executeNode(node parser.PlanTreeNode, tree *parser.PlanTree, ch chan int) {
	leftCh := make(chan int, 1)
	rightCh := make(chan int, 1)
	
	if node.Left != -1 {
		leftNode := tree.Nodes[node.Left]
		ok := false
		for {
			mutex.Lock()
			if leftNode.Status == 0 {
				leftNode.Status = 1
				mutex.Unlock()
				ok = true
				break
			} else if leftNode.Status == 1 {
				mutex.Unlock()
				time.Sleep(200 * time.Millisecond)
				continue
			} else {
				mutex.Unlock()
				leftCh <- 1
				break
			}
		}
		if ok {
			go executeNode(leftNode, tree, leftCh)
		}
	} else {
		leftCh <- 1
	}
	if node.Right != -1 {
		rightNode := tree.Nodes[node.Right]
		ok := false
		for {
			mutex.Lock()
			if rightNode.Status == 0 {
				rightNode.Status = 1
				mutex.Unlock()
				ok = true
				break
			} else if rightNode.Status == 1 {
				mutex.Unlock()
				time.Sleep(200 * time.Millisecond)
				continue
			} else {
				mutex.Unlock()
				rightCh <- 1
				break
			}
		} 
		
		if ok {
			go executeNode(rightNode, tree, rightCh)
		}
	} else {
		rightCh <- 1
	}

	rc := <- leftCh
	if rc == 0 {
		ch <- 0
		return
	}
	rc = <- rightCh
	if rc == 0 {
		ch <- 0
		return
	}

	eres := executeOperator(node, tree)
	if eres == 0 {
		ch <- 0
		return
	}

	mutex.Lock()
	tree.Nodes[node.Nodeid].Status = 2
	mutex.Unlock()

	ch <- 1
}
func DropTable(tableName string) {
	//清理main的tmptable
	connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
	client, _ := sql.Open("postgres", connStr)
	defer client.Close()
	// nodeType := node.NodeType
	// siteName := node.Locate
	// ServerName := storage.ServerName()

	// TODO: assert(plan_node.Right = -1)
	fmt.Println("main drop table:", tableName)
	sqlStr := "drop table if exists " + tableName

	stmt, _ := client.Prepare(sqlStr) //err
	defer stmt.Close()
	res, _ := stmt.Exec() //err
	println(res)

	// println(res)

}
func CleanTmpTable(node parser.PlanTreeNode) {
	//清理main的tmptable
	connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
	client, _ := sql.Open("postgres", connStr)
	defer client.Close()
	nodeType := node.NodeType
	siteName := node.Locate
	ServerName := storage.ServerName()

	if nodeType != 1 || (nodeType == 1 && siteName != ServerName) {
		tablename := node.TmpTable

		// TODO: assert(plan_node.Right = -1)
		fmt.Println("main drop table:", tablename)
		sqlStr := "drop table if exists " + tablename

		stmt, _ := client.Prepare(sqlStr) //err
		defer stmt.Close()
		res, _ := stmt.Exec() //err
		println(res)

		// println(res)
	}
}

func executeSP(node parser.PlanTreeNode, tree *parser.PlanTree) int {
	//连接数据库
	connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
	client, _ := sql.Open("postgres", connStr)
	defer client.Close()
	// fmt.Println("SP client:", err)

	var sqlStr string
	tableName := tree.Nodes[node.Left].TmpTable
	if node.ExecStmtCols == "" {
		sqlStr = "create table " + node.TmpTable + " as select * from " + tableName
	} else {

		sqlStr = "create table " + node.TmpTable + " as select " + node.ExecStmtCols + " from " + tableName

	}
	if node.ExecStmtWhere != "" {
		sqlStr += " " + node.ExecStmtWhere
		fmt.Println("wheresql:", sqlStr)
	}

	ServerName := storage.ServerName()
	if node.Locate == ServerName {

		// findQuery := "select * from " + node.TmpTable
		// fmt.Println("SP main", findQuery)
		// _, fres := client.Query(findQuery)
		// if fres != nil {
		stmt, _ := client.Prepare(sqlStr) //err
		defer stmt.Close()
		// fmt.Println("SP prepare:", err)
		res, _ := stmt.Exec() //err
		// fmt.Println("SP exec:", err)
		println(res)
		tableList1 = append(tableList1, node.TmpTable)
		// }

		// fmt.Println("main will drop", tree.Nodes[node.Left].TmpTable)
		// CleanTmpTable(tree.Nodes[node.Left])

	} else {
		fmt.Println("SP not main", sqlStr)
		address := storage.GetServerAddress(node.Locate)
		// leftAddr := storage.GetServerAddress(tree.Nodes[node.Left].Locate)

		// if tree.Nodes[node.Left].NodeType != 1 && tree.Nodes[node.Left].Left != -1 {
		// 	res1 := storage.ExecRemoteSql("drop table if exiss "+tableName+";", leftAddr)
		// 	fmt.Println(tree.Nodes[node.Left].Locate, "left drop", res1)
		// }
		// findQuery := "select count(*) from pg_class where relname = 'tablename';"
		// findQuery = strings.Replace(findQuery, "tablename", node.TmpTable, -1)
		// fmt.Println(findQuery)
		// fres := storage.ExecRemoteSelect(findQuery, address)
		// fmt.Println("fres==0:", fres == "0")
		// if fres == "0" {
		// getRemoteTmpTable(tree.Nodes[node.Left], leftAddr, address)
		fmt.Println(sqlStr, address)
		res := storage.ExecRemoteSql(sqlStr, address)
		if int(res) == 2 {
			return 0
		}
		fmt.Println(res)
		switch {
		case node.Locate == "segment1":
			tableList2 = append(tableList2, node.TmpTable)
			break
		case node.Locate == "segment2":
			tableList3 = append(tableList3, node.TmpTable)
			break
		case node.Locate == "segment3":
			tableList4 = append(tableList4, node.TmpTable)
			break
		}

	}

	// if !(tree.Nodes[node.Left].NodeType == 1 && tree.Nodes[node.Left].Left == -1) {
	// 	res3 := storage.ExecRemoteSql("drop table if exiss "+tableName+";", address)
	// 	fmt.Println(node.Locate, "left drop", res3)
	// }
	return 1

}

func executeScan(node parser.PlanTreeNode, tree *parser.PlanTree) int {
	//连接数据库
	connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
	client, _ := sql.Open("postgres", connStr)
	// fmt.Println("scan client:", err)
	defer client.Close()

	var sqlStr string
	tableName := tree.Nodes[node.Left].TmpTable

	sqlStr = "create table " + node.TmpTable + " as select * from " + tableName + " ;"
	ServerName := storage.ServerName()
	if node.Locate == ServerName {
		leftAddr := storage.GetServerAddress(tree.Nodes[node.Left].Locate)

		getTmpTable(tree.Nodes[node.Left], leftAddr)

		// leftAddress := storage.GetServerAddress(tree.Nodes[node.Left].Locate)

		// res1 := storage.ExecRemoteSql("drop table if exists "+tableName+";", leftAddress)
		// fmt.Println("left drop", res1)

		// fmt.Println("sel:", sqlStr)
		stmt, _ := client.Prepare(sqlStr) //err
		// fmt.Println("sel prepare:", err)
		defer stmt.Close()
		res, _ := stmt.Exec() //err
		// fmt.Println("sel exec:", err)
		println(res)
		tableList1 = append(tableList1, node.TmpTable)

		// CleanTmpTable(tree.Nodes[node.Left])

	} else {
		// fmt.Println(node.Locate, "child:", tree.Nodes[node.Left].Locate)

		address := storage.GetServerAddress(node.Locate)
		// fmt.Println(sqlStr, address)
		// leftAddr := storage.GetServerAddress(tree.Nodes[node.Left].Locate)

		// getRemoteTmpTable(tree.Nodes[node.Left], address, address)

		// res1 := storage.ExecRemoteSql("drop table if exiss "+tableName+";", address)
		// fmt.Println("left drop", res1)

		res := storage.ExecRemoteSql(sqlStr, address)
		if int(res) == 2 {
			return 0
		}
		// fmt.Println("scan exec remote", res)
		switch {
		case node.Locate == "segment1":
			tableList2 = append(tableList2, node.TmpTable)
			break
		case node.Locate == "segment2":
			tableList3 = append(tableList3, node.TmpTable)
			break
		case node.Locate == "segment3":
			tableList4 = append(tableList4, node.TmpTable)
			break
		}
		// res3 := storage.ExecRemoteSql("drop table if exist "+tableName+";", address)
		// fmt.Println("left drop", res3)

	}
	return 1

}

func executeUnion(node parser.PlanTreeNode, tree *parser.PlanTree) int {
	//连接数据库
	connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
	client, _ := sql.Open("postgres", connStr)
	// fmt.Println("union client:", err)
	defer client.Close()

	var sqlStr string
	leftTableName := tree.Nodes[node.Left].TmpTable
	rightTableName := tree.Nodes[node.Right].TmpTable
	sqlStr = "create table " + node.TmpTable + " as select * from " + leftTableName + " union all" + " select * from " + rightTableName + ";"

	ServerName := storage.ServerName()
	if node.Locate == ServerName {
		leftAddr := storage.GetServerAddress(tree.Nodes[node.Left].Locate)
		rightAddr := storage.GetServerAddress(tree.Nodes[node.Right].Locate)
		if tree.Nodes[node.Left].Locate != ServerName {
			leftTableName = strings.ToLower(leftTableName)
			findQuery := "select * from " + leftTableName
			_, fres := client.Query(findQuery)
			if fres != nil {
				getTmpTable(tree.Nodes[node.Left], leftAddr)
				tableList1 = append(tableList1, tree.Nodes[node.Left].TmpTable)
			}
			// getTmpTable(tree.Nodes[node.Left], leftAddr)
			// res1 := storage.ExecRemoteSql("drop table if exists "+leftTableName+";", leftAddr)
			// fmt.Println("left drop", res1)
		}

		if tree.Nodes[node.Right].Locate != ServerName {
			rightTableName = strings.ToLower(rightTableName)
			findQuery := "select * from " + rightTableName
			_, fres := client.Query(findQuery)
			if fres != nil {
				getTmpTable(tree.Nodes[node.Right], rightAddr)
				tableList1 = append(tableList1, tree.Nodes[node.Right].TmpTable)
			}

			// res2 := storage.ExecRemoteSql("drop table if exists "+rightTableName+";", rightAddr)
			// fmt.Println("right drop", res2)

		}

		fmt.Println("union:", sqlStr)
		stmt, _ := client.Prepare(sqlStr) //err
		// fmt.Println("union prepare:", err)
		defer stmt.Close()
		res, _ := stmt.Exec() //err
		// fmt.Println("union exec:", err)
		println(res)
		tableList1 = append(tableList1, node.TmpTable)

		// fmt.Println("main will drop", tree.Nodes[node.Left].TmpTable)
		// CleanTmpTable(tree.Nodes[node.Left])
		// fmt.Println("main will drop", tree.Nodes[node.Right].TmpTable)
		// CleanTmpTable(tree.Nodes[node.Right])
	} else {
		address := storage.GetServerAddress(node.Locate)
		leftAddr := storage.GetServerAddress(tree.Nodes[node.Left].Locate)
		rightAddr := storage.GetServerAddress(tree.Nodes[node.Right].Locate)
		if tree.Nodes[node.Left].Locate != node.Locate { //tree.Nodes[node.Left].NodeType != 1 ||
			findQuery := "select count(*) from pg_class where relname = 'tablename';"
			leftTableName = strings.ToLower(leftTableName)
			findQuery = strings.Replace(findQuery, "tablename", leftTableName, -1)

			fres := storage.ExecRemoteSelect(findQuery, address)
			if fres == "2" {
				return 0
			}
			// fmt.Println(fres == "(0)")
			// fmt.Println(fres == 0)
			if fres == "(0)" {
				getRemoteTmpTable(tree.Nodes[node.Left], leftAddr, address)
			}

			switch {
			case node.Locate == "segment1":
				tableList2 = append(tableList2, leftTableName)
				break
			case node.Locate == "segment2":
				tableList3 = append(tableList3, leftTableName)
				break
			case node.Locate == "segment3":
				tableList4 = append(tableList4, leftTableName)
				break
			}
			// res1 := storage.ExecRemoteSql("drop table if exiss "+leftTableName+";", leftAddr)
			// fmt.Println(tree.Nodes[node.Left].Locate, "left drop", res1)
		}

		// fmt.Println("right locage:", tree.Nodes[node.Right].Locate != node.Locate)
		if tree.Nodes[node.Right].Locate != node.Locate { //tree.Nodes[node.Right].NodeType != 1 ||
			findQuery2 := "select count(*) from pg_class where relname = 'tablename';"
			rightTableName = strings.ToLower(rightTableName)
			findQuery2 = strings.Replace(findQuery2, "tablename", rightTableName, -1)

			fres2 := storage.ExecRemoteSelect(findQuery2, address)
			if fres2 == "2" {
				return 0
			}
			// fmt.Println(fres2)
			// fmt.Println(fres2 == "(0)")
			// fmt.Println(fres == 0)
			if fres2 == "(0)" {
				getRemoteTmpTable(tree.Nodes[node.Right], rightAddr, address)
			}

			switch {
			case node.Locate == "segment1":
				tableList2 = append(tableList2, rightTableName)
				break
			case node.Locate == "segment2":
				tableList3 = append(tableList3, rightTableName)
				break
			case node.Locate == "segment3":
				tableList4 = append(tableList4, rightTableName)
				break
			}
			// res2 := storage.ExecRemoteSql("drop table if exists "+rightTableName+";", rightAddr)
			// fmt.Println(tree.Nodes[node.Right].Locate, "right drop", res2)
		}

		res := storage.ExecRemoteSql(sqlStr, address)
		if int(res) == 2 {
			return 0
		}
		// fmt.Println(res)
		// fmt.Println(leftTableName, rightTableName)

		switch {
		case node.Locate == "segment1":
			tableList2 = append(tableList2, node.TmpTable)
			break
		case node.Locate == "segment2":
			tableList3 = append(tableList3, node.TmpTable)
			break
		case node.Locate == "segment3":
			tableList4 = append(tableList4, node.TmpTable)
			break
		}

		// if !(tree.Nodes[node.Left].NodeType == 1 && tree.Nodes[node.Left].Left == -1) {
		// 	res3 := storage.ExecRemoteSql("drop table if exists "+leftTableName+";", address)
		// 	fmt.Println(node.Locate, "left drop", res3)
		// }
		// if !(tree.Nodes[node.Right].NodeType == 1 && tree.Nodes[node.Right].Left == -1) {
		// 	res4 := storage.ExecRemoteSql("drop table if exists "+rightTableName+";", address)
		// 	fmt.Println(node.Locate, "right drop", res4)
		// }

	}
	return 1

}
func getTmpTable(node parser.PlanTreeNode, address string) int {
	connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
	client, _ := sql.Open("postgres", connStr)
	defer client.Close()

	tableName := node.TmpTable
	// fmt.Println("tmpaddr", address)
	tableName = strings.ToLower(tableName)
	CreateSql := storage.GetRemoteSchema(tableName, address)

	CreateSql = strings.Replace(CreateSql, "integer(32)", "integer", -1)
	CreateSql = strings.Replace(CreateSql, "integer(64)", "integer", -1)
	CreateSql = strings.Replace(CreateSql, ", );", " );", -1)
	// fmt.Println("createsql", CreateSql, address)

	stmt, _ := client.Prepare(CreateSql)
	defer stmt.Close()
	// fmt.Println("tmpcreate prepare", err)
	res, _ := stmt.Exec()
	println(res)
	// fmt.Println("tmp create", res, err)

	insertQuery := "insert into " + tableName + " values "
	query := "select * from " + tableName + " ;"
	// fmt.Println("before exec remote select", query)
	insertPlus := storage.ExecRemoteSelect(query, address)
	if insertPlus == "2" {
		return 0
	}
	// fmt.Println("after")

	insertQuery += insertPlus + ";"

	stmt2, _ := client.Prepare(insertQuery)
	defer stmt2.Close()
	// fmt.Println("tmpinsert prepare", err)
	res2, _ := stmt2.Exec()
	println(res2)
	// fmt.Println("[trans amount]", res2, err)
	return 1

}

func getRemoteTmpTable(node parser.PlanTreeNode, address string, dest string) int {

	tableName := node.TmpTable

	tableName = strings.ToLower(tableName)
	// fmt.Println(tableName)
	CreateSql := storage.GetRemoteSchema(tableName, address)
	if CreateSql == "2" {
		return 0
	}

	CreateSql = strings.Replace(CreateSql, "integer(32)", "integer", -1)
	CreateSql = strings.Replace(CreateSql, "integer(64)", "integer", -1)
	CreateSql = strings.Replace(CreateSql, ", );", " );", -1)
	// fmt.Println("rtmptable", CreateSql)

	res := storage.ExecRemoteSql(CreateSql, dest)
	if int(res) == 2 {
		return 0
	}
	// fmt.Println("tmp,", res)

	insertQuery := "insert into " + tableName + " values "
	query := "select * from " + tableName + " ;"
	// fmt.Println("get insert")
	insertPlus := storage.ExecRemoteSelect(query, address)
	if insertPlus == "2" {
		return 0
	}
	// fmt.Println("get done insert", insertPlus[:300])
	insertQuery += insertPlus + ";"
	res2 := storage.ExecRemoteSql(insertQuery, dest)
	if int(res2) == 2 {
		return 0
	}
	// fmt.Println("[trans amount]", res2)
	return 1
}
func executeJoin(node parser.PlanTreeNode, tree *parser.PlanTree) int {
	//连接数据库
	connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
	client, _ := sql.Open("postgres", connStr)
	// fmt.Println("join client:", err)
	defer client.Close()

	var sqlStr string
	leftTableName := tree.Nodes[node.Left].TmpTable
	rightTableName := tree.Nodes[node.Right].TmpTable

	print(node.ExecStmtWhere, node.Where)
	Cols := "*"
	if node.ExecStmtCols != "" {
		Cols = node.ExecStmtCols
	}
	// fmt.Println("cols:", Cols)
	if node.ExecStmtWhere == "" {
		sqlStr = "create table " + node.TmpTable + " as select " + Cols + " from " + leftTableName + " natural join " + rightTableName + ";"

	} else {
		sqlStr = "create table " + node.TmpTable + " as select " + Cols + " from " + leftTableName + " inner join " + rightTableName + " " + " on " + strings.TrimPrefix(node.ExecStmtWhere, "where") + ";"

	}

	// fmt.Println("join sql", sqlStr, node.Locate, ServerName)
	ServerName := storage.ServerName()
	if node.Locate == ServerName {
		// fmt.Println("join main")
		leftAddr := storage.GetServerAddress(tree.Nodes[node.Left].Locate)
		rightAddr := storage.GetServerAddress(tree.Nodes[node.Right].Locate)
		// fmt.Println("join addr:", leftAddr, rightAddr)
		if tree.Nodes[node.Left].Locate != ServerName {
			leftTableName = strings.ToLower(leftTableName)
			findQuery := "select * from " + leftTableName
			_, fres := client.Query(findQuery)
			if fres != nil {
				getTmpTable(tree.Nodes[node.Left], leftAddr)

				tableList1 = append(tableList1, tree.Nodes[node.Left].TmpTable)
			}

			// res1 := storage.ExecRemoteSql("drop table if exists "+leftTableName+";", leftAddr)
			// fmt.Println("left drop", res1)
		}
		if tree.Nodes[node.Right].Locate != ServerName {
			rightTableName = strings.ToLower(rightTableName)
			findQuery := "select * from " + rightTableName
			_, fres := client.Query(findQuery)
			// fmt.Println("join in:", findQuery)
			if fres != nil {
				getTmpTable(tree.Nodes[node.Right], rightAddr)

				tableList1 = append(tableList1, tree.Nodes[node.Right].TmpTable)
			}

			// res2 := storage.ExecRemoteSql("drop table if exists "+rightTableName+";", rightAddr)
			// fmt.Println("right drop", res2)

		}

		fmt.Println("join:", sqlStr)
		stmt, err := client.Prepare(sqlStr) //err
		fmt.Println("join prepare:", err)
		defer stmt.Close()
		res, _ := stmt.Exec() //err
		// fmt.Println("join exec:", err)
		println(res)
		tableList1 = append(tableList1, node.TmpTable)
		// fmt.Println("main will drop", tree.Nodes[node.Left].TmpTable)
		// CleanTmpTable(tree.Nodes[node.Left])
		// fmt.Println("main will drop", tree.Nodes[node.Right].TmpTable)
		// CleanTmpTable(tree.Nodes[node.Right])
	} else {
		// fmt.Println("join not main")

		address := storage.GetServerAddress(node.Locate)
		leftAddr := storage.GetServerAddress(tree.Nodes[node.Left].Locate)
		rightAddr := storage.GetServerAddress(tree.Nodes[node.Right].Locate)
		// fmt.Println("join addr:", address, leftAddr, rightAddr)
		if tree.Nodes[node.Left].Locate != node.Locate { //tree.Nodes[node.Left].NodeType != 1 ||
			findQuery := "select count(*) from pg_class where relname = 'tablename';"
			leftTableName = strings.ToLower(leftTableName)
			findQuery = strings.Replace(findQuery, "tablename", leftTableName, -1)

			fres := storage.ExecRemoteSelect(findQuery, address)
			if fres == "2" {
				return 0
			}

			if fres == "(0)" {
				// fmt.Println("fres==0 left:", leftTableName)
				getRemoteTmpTable(tree.Nodes[node.Left], leftAddr, address)
				switch {
				case node.Locate == "segment1":
					tableList2 = append(tableList2, leftTableName)
					break
				case node.Locate == "segment2":
					tableList3 = append(tableList3, leftTableName)
					break
				case node.Locate == "segment3":
					tableList4 = append(tableList4, leftTableName)
					break
				}

				// res1 := storage.ExecRemoteSql("drop table if exiss "+leftTableName+";", leftAddr)
				// fmt.Println(tree.Nodes[node.Left].Locate, "left drop", res1)
			}

		}

		if tree.Nodes[node.Right].Locate != node.Locate { //tree.Nodes[node.Right].NodeType != 1 ||
			findQuery2 := "select count(*) from pg_class where relname = 'tablename';"
			rightTableName = strings.ToLower(rightTableName)
			findQuery2 = strings.Replace(findQuery2, "tablename", rightTableName, -1)

			fres2 := storage.ExecRemoteSelect(findQuery2, address)
			if fres2 == "2" {
				return 0
			}
			// fmt.Println(fres2)
			if fres2 == "(0)" {
				// fmt.Println("fres==0 right:", rightTableName)
				getRemoteTmpTable(tree.Nodes[node.Right], rightAddr, address)
				switch {
				case node.Locate == "segment1":
					tableList2 = append(tableList2, rightTableName)
					break
				case node.Locate == "segment2":
					tableList3 = append(tableList3, rightTableName)
					break
				case node.Locate == "segment3":
					tableList4 = append(tableList4, rightTableName)
					break
				}
				// res1 := storage.ExecRemoteSql("drop table if exiss "+rightTableName+";", rightAddr)
				// fmt.Println(tree.Nodes[node.Right].Locate, "right drop", res1)
			}
		}

		res := storage.ExecRemoteSql(sqlStr, address)
		if int(res) == 2 {
			return 0
		}
		// fmt.Println(res)
		switch {
		case node.Locate == "segment1":
			tableList2 = append(tableList2, node.TmpTable)
			break
		case node.Locate == "segment2":
			tableList3 = append(tableList3, node.TmpTable)
			break
		case node.Locate == "segment3":
			tableList4 = append(tableList4, node.TmpTable)
			break
		}
		return 1
		// if !(tree.Nodes[node.Left].NodeType == 1 && tree.Nodes[node.Left].Left == -1) {
		// 	res3 := storage.ExecRemoteSql("drop table if exists "+leftTableName+";", address)
		// 	fmt.Println(node.Locate, "left drop", res3)
		// }
		// if !(tree.Nodes[node.Right].NodeType == 1 && tree.Nodes[node.Right].Left == -1) {
		// 	res4 := storage.ExecRemoteSql("drop table if exists "+rightTableName+";", address)
		// 	fmt.Println(node.Locate, "right drop", res4)
		// }
	}

	// client.Close()
	return 1
}

func generateCreateQuery(node parser.PlanTreeNode) string {
	//连接数据库
	connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
	client, _ := sql.Open("postgres", connStr)
	// fmt.Println("create client:", err)
	defer client.Close()

	var create_sql sql.NullString
	query := "select showcreatetable('public','table_name');"
	query = strings.Replace(query, "table_name", node.TmpTable, -1)
	// fmt.Println("create query", query)

	rows, _ := client.Query(query)
	defer rows.Close()
	// fmt.Println("create query:", err)
	rows.Next()
	_ = rows.Scan(&create_sql)
	// fmt.Println("createScan err:", err)
	// fmt.Println(create_sql.String)
	createSql := create_sql.String
	createSql = strings.Replace(createSql, "integer(32)", "integer", -1)
	createSql = strings.Replace(createSql, "integer(64)", "integer", -1)
	createSql = strings.Replace(createSql, ", );", " );", -1)
	createSql = createSql[1:]
	// fmt.Println(createSql)

	return createSql
}

func generateInsertQuery(node parser.PlanTreeNode) ([]string, bool) {
	//连接数据库
	connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
	client, _ := sql.Open("postgres", connStr)
	// fmt.Println("insert client:", err)
	defer client.Close()

	mySlice := make([]string, 0)
	insert_query := "insert into " + node.TmpTable + " values "
	query := "select * from " + node.TmpTable
	// println(query)
	rows, _ := client.Query(query) //err:_

	colTypes, _ := rows.ColumnTypes()

	types := make([]reflect.Type, len(colTypes))
	for i, tp := range colTypes {
		// ScanType
		scanType := tp.ScanType()
		types[i] = scanType
	}
	// fmt.Println(" ")
	values := make([]interface{}, len(colTypes))
	for i := range values {
		values[i] = reflect.New(types[i]).Interface()
	}
	i := 0
	for rows.Next() {
		if i%1000 == 0 && i != 0 {
			insert_query = insert_query + ";"
			mySlice = append(mySlice, insert_query)
			insert_query = "insert into " + node.TmpTable + " values "
		} else if i != 0 {
			insert_query = insert_query + ", "
		}
		_ = rows.Scan(values...) //err

		insert_query = insert_query + "("
		for j := range values {
			if j != 0 {
				insert_query = insert_query + ", "
			}
			value := reflect.ValueOf(values[j]).Elem().Interface()
			insert_query = insert_query + Strval(value)
			// fmt.Print(Strval(value))
			// fmt.Print(" ")
		}
		insert_query = insert_query + ")"
		// fmt.Println(" ")
		i++
	}
	insert_query = insert_query + ";"
	mySlice = append(mySlice, insert_query)
	// client.Close()
	if i == 0 {
		return mySlice, false
	} else {
		return mySlice, true
	}
}

// select * from
// ExecRemoteSelect(select ) Ex
// ExecGetTable("")
func executeRemoteCreateStmt(address string, create_sql string) {
	// fmt.Println("executeRemoteCreateStmt")
	res := storage.ExecRemoteSql(create_sql, address)
	fmt.Println("exec remote", res)
}
func executeTrans(node parser.PlanTreeNode) {
	ServerName := storage.ServerName()
	// ServerName := "main"
	fmt.Println("servername", ServerName)
	fmt.Println("nodeserver", node.Locate)
	if node.Locate != ServerName {
		fmt.Println("node locate", node.Locate)
		address := storage.GetServerAddress(node.Locate)
		// address := node.Locate + ":" + node.Dest
		create_sql := generateCreateQuery(node)
		executeRemoteCreateStmt(address, create_sql)
		insert_query, success := generateInsertQuery(node)

		if success {
			for _, query := range insert_query {
				executeRemoteCreateStmt(address, query)
			}
		}

	}
}

func executeOperator(node parser.PlanTreeNode, tree *parser.PlanTree) int {
	if node.NodeType == 2 || node.NodeType == 3 { //"scan" "projection"{
		return executeSP(node, tree)

	} else if node.NodeType == 4 { //strings.HasPrefix(node.NodeType, "join") {
		return executeJoin(node, tree)

	} else if node.NodeType == 5 { //"union" {
		return executeUnion(node, tree)

	} else if node.NodeType == 1 && node.Left != -1 {
		executeScan(node, tree)
	}
	// executeTrans(node)
	return 1
}

func printResult(tree *parser.PlanTree) []string {
	result := make([]string, 0)
	if tree.Nodes[tree.Root].Locate == "main" {
		connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
		client, _ := sql.Open("postgres", connStr)
		defer client.Close()

		// var result string
		node := tree.Nodes[tree.Root]

		query := "select * from " + node.TmpTable
		println("PrintResult:", query)
		rows, _ := client.Query(query)
		// fmt.Println(rows)
		colTypes, _ := rows.ColumnTypes()

		types := make([]reflect.Type, len(colTypes))
		for i, tp := range colTypes {
			// ScanType
			scanType := tp.ScanType()
			types[i] = scanType
		}
		// fmt.Println(" ")
		values := make([]interface{}, len(colTypes))
		for i := range values {
			values[i] = reflect.New(types[i]).Interface()
		}
		i := 0
		for rows.Next() {
			var res string
			// todo: 只插入前100条，之后需要修改
			// if i > 10 {
			// 	break
			// }
			// todo: 只插入前100条，之后需要修改
			_ = rows.Scan(values...)

			// fmt.Print("|")
			res += "|"
			for j := range values {

				value := reflect.ValueOf(values[j]).Elem().Interface()
				// fmt.Print(Strval(value))
				res += Strval(value)
				// fmt.Print("|")
				res += "|"
			}
			// fmt.Println(" ")
			res += " "
			// result += res + "\n"
			result = append(result, res)
			i++
		}
		CleanTmpTable(node)
	} else {
		address := storage.GetServerAddress(tree.Nodes[tree.Root].Locate)
		node := tree.Nodes[tree.Root]

		query := "select * from " + node.TmpTable

		values := storage.ExecRemoteSelect(query, address)
		rows := strings.Split(values, "),")
		i := 0
		for _, row_val := range rows {
			var res string
			// todo: 只插入前100条，之后需要修改
			// if i > 10 {
			// 	break
			// }
			// todo: 只插入前100条，之后需要修改

			// fmt.Print("|")
			res += "|"
			val := strings.Split(row_val, ",")
			for ind, j := range val {
				if ind == 0 {
					j = j[1:]
				} else if ind == len(val)-1 {
					j = j[:len(j)-1]
				}
				// value := reflect.ValueOf(values[j]).Elem().Interface()
				// fmt.Print(Strval(value))
				res += Strval(j)
				// fmt.Print("|")
				res += "|"
			}
			// fmt.Println(" ")
			res += " "
			// result += res + "\n"
			result = append(result, res)
			i++
		}

	}

	return result
}
func printTree(node parser.PlanTreeNode, tree *parser.PlanTree, num int32) {
	fmt.Println(node.TmpTable)
	if node.Left != -1 {
		leftNode := tree.Nodes[node.Left]
		fmt.Println("left", leftNode.TmpTable)
		printTree(leftNode, tree, num+1)
	} else {
		fmt.Println("no left node")
	}

	if node.Right != -1 {
		rightNode := tree.Nodes[node.Left]
		fmt.Println("right", rightNode.TmpTable)
		printTree(rightNode, tree, num+1)
	} else {
		fmt.Println("no right node")
	}

}
func CleanAllTable() {
	for _, table := range tableList1 {
		// continue
		DropTable(table)
	}

	addr := storage.GetServerAddress("segment1")
	for _, table := range tableList2 {
		res1 := storage.ExecRemoteSql("drop table if exists "+table+";", addr)
		fmt.Println(res1)
	}

	addr = storage.GetServerAddress("segment2")
	for _, table := range tableList3 {
		res1 := storage.ExecRemoteSql("drop table if exists "+table+";", addr)
		fmt.Println(res1)
	}

	addr = storage.GetServerAddress("segment3")
	for _, table := range tableList4 {
		res1 := storage.ExecRemoteSql("drop table if exists "+table+";", addr)
		fmt.Println(res1)
	}
}
func ExecuteInsert(tree *parser.PlanTree) int {
	insertNum := 0
	nodeNum := int(tree.NodeNum)
	for i := 1; i <= nodeNum; i++ {
		node := tree.Nodes[i]

		insertSql := "insert into " + node.TmpTable + " (" + node.ExecStmtCols + ") values (" + node.Cols + ")"
		if node.Locate != "main" {
			address := storage.GetServerAddress(node.Locate)
			res2 := storage.ExecRemoteSql(insertSql, address)
			insertNum += res2
			fmt.Println("remote insert", res2)
		} else {
			connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
			client, _ := sql.Open("postgres", connStr)
			defer client.Close()
			// fmt.Println("insert err", err)
			// stmt2, _ := client.Prepare(insertSql)
			// fmt.Println("insert prepare", err)
			res2, _ := client.Exec(insertSql)
			fmt.Println("insert", res2)
			res2num, _ := res2.RowsAffected()
			insertNum += int(res2num)
			// fmt.Println("insert", res2)
		}
	}
	return insertNum
}
func ExecuteDelete(tree *parser.PlanTree) int {
	delNum := [4]int{0, 0, 0, 0}
	nodeNum := int(tree.NodeNum)
	ch := make(chan int, 1)
	for i := 1; i <= nodeNum; i++ {
		node := tree.Nodes[i]
		fmt.Println(node.TmpTable, node.ExecStmtWhere)
		if node.Left != -1 {
			executeNode(tree.Nodes[node.Left], tree, ch)
			<- ch
		}

		delSql := "delete from " + node.TmpTable + " " + node.ExecStmtWhere

		// if node.ExecStmtWhere != "" {
		// 	delSql += "  " + node.ExecStmtWhere
		// }
		fmt.Println(delSql)
		if node.Locate == "main" {
			connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
			client, _ := sql.Open("postgres", connStr)
			defer client.Close()
			// fmt.Println("delete err", err)
			stmt2, _ := client.Prepare(delSql)
			// fmt.Println("delete prepare", err)
			res2, _ := stmt2.Exec()
			res2num, _ := res2.RowsAffected()
			delNum[0] += int(res2num)
			// fmt.Println("delete", res2, err)
		} else {
			address := storage.GetServerAddress(node.Locate)
			fmt.Println(address)
			res2 := storage.ExecRemoteSql(delSql, address)
			switch {
			case node.Locate == "segment1":
				delNum[1] += res2
				break
			case node.Locate == "segment2":
				delNum[2] += res2
				break
			case node.Locate == "segment3":
				delNum[3] += res2
				break
			}
			// delNum += res2
			// fmt.Println("remote delete", res2)
		}

	}
	ans := 0
	if tree.Nodes[tree.Root].TmpTable == "customer" {
		ans = delNum[0]
		for i := 1; i < len(delNum); i++ {
			if delNum[i] > ans {
				ans = delNum[i]
			}
		}
	} else {
		ans = delNum[0] + delNum[1] + delNum[2] + delNum[3]
	}
	return ans
}
func ExecuteCreateDB(tree *parser.PlanTree) {
	nodeNum := int(tree.NodeNum)
	for i := 1; i <= nodeNum; i++ {
		node := tree.Nodes[i]

		createSql := "create database " + node.TmpTable
		if node.Locate != "main" {
			address := storage.GetServerAddress(node.Locate)
			res2 := storage.ExecRemoteSql(createSql, address)
			fmt.Println("remote insert", res2)
		} else {
			connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
			client, _ := sql.Open("postgres", connStr)
			defer client.Close()
			// fmt.Println("insert err", err)
			stmt2, _ := client.Prepare(createSql)
			// fmt.Println("insert prepare", err)
			res2, _ := stmt2.Exec()
			fmt.Println("insert", res2)
		}
	}
}

func CreateTable(tree *parser.PlanTree) {
	nodeNum := int(tree.NodeNum)
	for i := 1; i <= nodeNum; i++ {
		node := tree.Nodes[i]

		createSql := node.ExecStmtWhere
		createSql = strings.Replace(createSql, "key", "primary key", -1)
		fmt.Println(createSql)
		if node.Locate != "main" {

			address := storage.GetServerAddress(node.Locate)
			res2 := storage.ExecRemoteSql(createSql, address)
			fmt.Println("remote insert", res2)

		} else {
			connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
			client, _ := sql.Open("postgres", connStr)
			defer client.Close()
			// fmt.Println("insert err", err)
			stmt2, _ := client.Prepare(createSql)
			// fmt.Println("insert prepare", err)
			res2, _ := stmt2.Exec()
			fmt.Println("insert", res2)
		}
	}
	t := tree.TableMeta
	err := storage.StoreTableMeta(t)
	fmt.Println(err)
}

func GetSites(tree *parser.PlanTree) string {
	maps := make(map[string]int, 0)
	for i := 1; i <= int(tree.NodeNum); i++ {
		maps[tree.Nodes[i].Locate] = 1
	}

	ans := ""
	for k := range maps {
		if k != "" {
			ans += k + ","
		}
	}
	ans = strings.TrimSuffix(ans, ",")

	return ans
}

func ExecuteDropTable(tree *parser.PlanTree) int {
	nodeNum := int(tree.NodeNum)
	dropNum := 0

	for i := 1; i <= nodeNum; i++ {
		node := tree.Nodes[i]

		createSql := "drop table " + node.TmpTable
		if node.Locate != "main" {
			address := storage.GetServerAddress(node.Locate)
			res2 := storage.ExecRemoteSql(createSql, address)
			dropNum += res2
			fmt.Println("drop table", res2)
		} else {
			connStr := fmt.Sprintf("dbname=%s port=%d user=%s password=%s sslmode=%s", db.dbname, db.port, db.user, db.password, db.sslmode)
			client, _ := sql.Open("postgres", connStr)
			defer client.Close()
			// fmt.Println("insert err", err)
			stmt2, _ := client.Prepare(createSql)
			// fmt.Println("insert prepare", err)
			res2, _ := stmt2.Exec()
			res2num, _ := res2.RowsAffected()
			dropNum += int(res2num)
			// println(res2)
			// fmt.Println("insert", res2, err)
		}
	}
	err := storage.DropTableMeta(tree.Nodes[1].TmpTable)
	println(err)
	return dropNum
}
func Execute(tree *parser.PlanTree) (string, int) {
	// printTree(tree.Nodes[tree.Root], tree, 0)
	// tree.Print()
	tree.Print()
	resultStr := ""
	resultLen := 0
	ch := make(chan int, 1)
	if len(tree.Nodes) == 0 {
		return "empty tree", -1
	}

	if tree.Nodes[tree.Root].NodeType == 6 {
		resultLen = ExecuteInsert(tree)
		resultStr = "insert success\n"
		// resultLen = 0
	} else if tree.Nodes[tree.Root].NodeType == 7 {
		// tree.Print()
		fmt.Println("delete execute")
		resultLen = ExecuteDelete(tree)
		resultStr = "delete success\n"
		// resultLen = 0
	} else if tree.Nodes[tree.Root].NodeType == 8 {
		// tree.Print()
		ExecuteCreateDB(tree)
		fmt.Println("create db success")
		resultStr = "db success\n"
		resultLen = 0
	} else if tree.Nodes[tree.Root].NodeType == 10 {
		// tree.Print()
		CreateTable(tree)
		fmt.Println("create table success")
		resultStr = "table success\n"
		resultLen = 0
	} else if tree.Nodes[tree.Root].NodeType == 11 {
		tableNameList := storage.GetAllTableMetas()
		for _, table := range tableNameList {
			fmt.Println(table)
			tableName := table.TableName
			fragNum := table.FragNum
			fragSchema := table.FragSchema

			resultStr += "Table: " + tableName + "\n"
			resultStr += "fragNum: " + strconv.Itoa(fragNum) + "\n"
			for _, tableInfo := range fragSchema {
				resultStr += "-------\n"
				site := tableInfo.SiteName
				Cols := tableInfo.Cols
				Conditions := tableInfo.Conditions
				resultStr += "sitename: " + site + "\n"
				resultStr += "colname: "
				for _, c := range Cols {
					resultStr += c + " "
				}
				resultStr += "\n"
				resultStr += "conditions: "
				for _, s := range Conditions {
					resultStr += s.Col + " " + s.Type + s.Comp + s.Value + ";"
				}
				resultStr += "\n"
				tableCount := storage.GetTableCount(tableName, site)
				resultStr += "count: " + strconv.Itoa(tableCount) + "\n"

			}
			resultStr += "=======\n"
		}
		// tableCount := storage.GetTableCount()
	} else if tree.Nodes[tree.Root].NodeType == 12 {
		fmt.Println("drop execute")
		resultLen = ExecuteDropTable(tree)
		resultStr = "drop success\n"
	} else {
		executeNode(tree.Nodes[tree.Root], tree, ch)
		res := <- ch
		if res == 0 {
			resultLen = 0
			resultStr = "connect error\n"
			return resultStr, resultLen
		}
		result := printResult(tree)
		resultLen = len(result)
		// var resultStr string
		i := 0
		for _, a := range result {
			if i > 10 {
				break
			}
			resultStr += a + "\n"

			i += 1

		}
		CleanAllTable()

		// fmt.Println(tableList1)
		// fmt.Println(tableList2)
		// fmt.Println(tableList3)
		// fmt.Println(tableList4)
		tableList1 = make([]string, 0)
		tableList2 = make([]string, 0)
		tableList3 = make([]string, 0)
		tableList4 = make([]string, 0)
		// tableList1 = []
		// tableList2 = []
		// tableList3 = []
		// tableList4 = []

		// address := storage.GetServerAddress("segment1")
		// findQuery := "select count(*) from pg_class where relname = 'tablename';"
		// findQuery = strings.Replace(findQuery, "tablename", "book", -1)
		// fres := storage.ExecRemoteSelect(findQuery, address)
		// fmt.Println(fres, fres == "(0)", fres == "(1)")
	}
	// tree.Print()
	fmt.Println("resultStr", resultStr)
	return resultStr, resultLen
}
