package hlr

import (
	"database/sql"
	"errors"
	"fmt"
)

var dbConn *sql.DB

//AppConnector hlr对数据库操作的上下文
type AppConnector struct {
	dbConn *sql.DB
}

//数据库连接
var appConnector *AppConnector

var createDatabaseSQL string = "CREATE DATABASE hlr;"

var createDomainSQL string = `CREATE TABLE hlr_domain (
	id serial PRIMARY KEY NOT NULL,
	name varchar(128) NOT NULL,
	tenant_id int NOT NULL,
	company text,
	enable boolean NOT NULL,
	update timestamp(6) DEFAULT now()
 	);`

var createGroupSQL string = `CREATE TABLE hlr_group (
	id serial PRIMARY KEY NOT NULL,
	name varchar(64) NOT NULL,
	group_desc text,
	parent_id int NULL,
	domain_id int NOT NULL,
	update timestamp(6) DEFAULT now()
	);`

var createUserSQL string = `CREATE TABLE hlr_user (
	id serial PRIMARY KEY NOT NULL,
	username varchar(64) NOT NULL,
	password varchar(64) NOT NULL,
	group_id int NOT NULL,
	update timestamp(6) DEFAULT now()
	);`

var insertDomainSQL string = "INSERT INTO hlr_domain(name,tenant_id,company,enable) VALUES('%s','%d','%s',%s) RETURNING id"
var insertGroupSQL string = "INSERT INTO hlr_group(name,group_desc,parent_id,domain_id) VALUES('%s','%s','%d','%d') RETURNING id"
var insertUserSQL string = "INSERT INTO hlr_user(username,password,group_id) VALUES('%s','%s','%d') RETURNING id"

var selectEnabledUsersSQL string = "select u.id,u.username,u.password,u.group_id from hlr_user as u left join hlr_group as g on u.group_id=g.id where g.domain_id=%d;"
var selectEnabledDomainsSQL string = "SELECT id,name,tenant_id,company FROM hlr_domain where enable=true"
var selectDomainSQL string = "SELECT id,name,tenant_id,company,enable FROM hlr_domain where id=%s"
var selectGroupSQL string = "SELECT id,name,group_desc,parent_id,domain_id FROM hlr_group where id=%s"
var selectUserSQL string = "SELECT id,username,password,group_id FROM hlr_user where id=%s"

/*
var updateDomainSQL string = "UPDATE hlr_domain SET name='%s',tenant_id=%d,company='%s',enable=%t where id=%d"
var updateGroupSQL string = "UPDATE hlr_group SET name='%s',group_desc='%s',parent_id=%d,domain_id=%d where id=%d"
var updateUserSQL string = "UPDATE hlr_user SET username='%s',password='%s',group_id=%d,status='%s',state='%s' where id=%d"
*/
var updateDomainSQL string = "UPDATE hlr_domain SET name=$1,tenant_id=$2,company=$3,enable=$4 where id=$5"
var updateGroupSQL string = "UPDATE hlr_group SET name=$1,group_desc=$2,parent_id=$3,domain_id=$4 where id=$5"
var updateUserSQL string = "UPDATE hlr_user SET username=$1,password=$2,group_id=$3 where id=$4"

var deleteDomainSQL string = "DELETE FROM hlr_domain WHERE id = $1"
var deleteGroupSQL string = "DELETE FROM hlr_group WHERE id = $1"
var deleteUserSQL string = "DELETE FROM hlr_user WHERE id = $1"

//GetDBConnector 返回与数据库的连接
func GetDBConnector() (*AppConnector, error) {
	if appConnector != nil {
		return appConnector, nil
	}
	return nil, errors.New("HLR is not connected to the database")
}

//OpenDBConnector 建立与DB的连接并返回
func OpenDBConnector(host string, port string, user string, password string, dbName string) (*AppConnector, error) {
	if appConnector != nil {
		return nil, errors.New("HLR already connected to the database")
	}
	dbSource := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbName)
	//dbSource := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable&dbname=%s", user, password, addr, user, dbName)
	dbConn, err := sql.Open("postgres", dbSource)
	if err != nil {
		return nil, err
	}
	err = dbConn.Ping()
	if err != nil {
		return nil, err
	}
	Info.Println("Successfully connected database!")
	appConnector = new(AppConnector)
	appConnector.dbConn = dbConn
	return appConnector, nil
}

//CloseDBConnector 关闭DB连接
func (c *AppConnector) CloseDBConnector() {
	c.dbConn.Close()
	c = nil
}

//execSQL 执行sql create update delete语句
func (c *AppConnector) execSQL(SQL string, args ...interface{}) error {
	stmt, err := c.dbConn.Prepare(SQL)
	if err != nil {
		//Error.Println("execSQL Prepare ", err)
		return err
	}
	result, err := stmt.Exec(args...)
	if err != nil {
		//Error.Println("execSQL stmt.exec ", err)
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		//Error.Println("execSQL RowsAffected ", err)
		return err
	}
	return nil
}

//insertSQL 执行sql插入语句 获取自增id
func (c *AppConnector) insertSQL(SQL string, args ...interface{}) (int64, error) {
	var id int64
	insertSQL := fmt.Sprintf(SQL, args...)
	err := c.dbConn.QueryRow(insertSQL).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

//CreateTable 创建表
func (c *AppConnector) CreateTable() {
	err := c.execSQL(createDomainSQL)
	if err != nil {
		Error.Println("create hlr_domain fail ", err)
	}
	err = c.execSQL(createGroupSQL)
	if err != nil {
		Error.Println("create hlr_group fail ", err)
	}
	err = c.execSQL(createUserSQL)
	if err != nil {
		Error.Println("create hlr_user fail ", err)
	}
}

//InsertDomain 插入一条域数据
func (c *AppConnector) InsertDomain(name string, tenantID int, company string, enable string) (int64, error) {
	id, err := c.insertSQL(insertDomainSQL, name, tenantID, company, enable)
	if err != nil {
		Error.Println(err)
		return 0, err
	}
	return id, nil
}

//InsertGroup 插入一条组数据
func (c *AppConnector) InsertGroup(name string, desc string, parentID int, domainID int) (int64, error) {
	id, err := c.insertSQL(insertGroupSQL, name, desc, parentID, domainID)
	if err != nil {
		Error.Println(err)
		return 0, err
	}
	return id, nil
}

//InsertUser 插入一条号码数据
func (c *AppConnector) InsertUser(user string, password string, groupID int) (int64, error) {
	id, err := c.insertSQL(insertUserSQL, user, password, groupID)
	if err != nil {
		Error.Println(err)
		return 0, err
	}
	return id, nil
}

//ReadDomain 查询domain信息
func (c *AppConnector) ReadDomain(id string) (*DomainInfo, error) {
	if c == nil {
		Error.Println("db context is null")
		return nil, errors.New("db context is null")
	}
	queryStr := fmt.Sprintf(selectDomainSQL, id)
	//Error.Println("-----queryStr----", queryStr)
	rows, err := c.dbConn.Query(queryStr)

	if err != nil {
		Error.Println(err.Error())
		return nil, err
	}
	defer rows.Close()
	p := new(DomainInfo)
	for rows.Next() {
		err := rows.Scan(&p.id, &p.Name, &p.TenantID, &p.Company, &p.Enable)
		if err != nil {
			Error.Println(err)
		}
		Error.Println(p.id, p.Name, p.TenantID, p.Company, p.Enable)
	}
	return p, nil
}

//ReadGroup 查询group信息
func (c *AppConnector) ReadGroup(id string) (*GroupInfo, error) {
	queryStr := fmt.Sprintf(selectGroupSQL, id)

	rows, err := c.dbConn.Query(queryStr)

	if err != nil {
		Error.Println(err.Error())
		return nil, err
	}
	defer rows.Close()
	p := new(GroupInfo)
	for rows.Next() {
		err := rows.Scan(&p.id, &p.Name, &p.GroupDesc, &p.ParentID, &p.DomainID) //id,name,desc,parent,domain
		if err != nil {
			Error.Println(err)
		}
		Error.Println(p.id, p.Name, p.GroupDesc, p.ParentID, p.DomainID)
	}
	return p, nil
}

//ReadUser 查询user信息
func (c *AppConnector) ReadUser(id string) (*UserInfo, error) {
	queryStr := fmt.Sprintf(selectUserSQL, id)

	rows, err := c.dbConn.Query(queryStr)

	if err != nil {
		Error.Println("ReadUser db query ", err)
		return nil, err
	}
	defer rows.Close()
	p := new(UserInfo)
	for rows.Next() {
		err := rows.Scan(&p.id, &p.Username, &p.Password, &p.GroupID)
		if err != nil {
			Error.Println(err)
		}
		Error.Println(p.id, p.Username, p.Password, p.GroupID)
	}
	return p, nil
}

//UpdateDomain 更新域信息
func (c *AppConnector) UpdateDomain(p *DomainInfo) error {
	err := c.execSQL(updateDomainSQL, p.Name, p.TenantID, p.Company, p.Enable, p.id)
	if err != nil {
		Error.Println(err)
		return err
	}
	return nil
}

//UpdateGroup 更新组信息
func (c *AppConnector) UpdateGroup(p *GroupInfo) error {
	err := c.execSQL(updateGroupSQL, p.Name, p.GroupDesc, p.ParentID, p.DomainID, p.id)
	if err != nil {
		Error.Println(err)
		return err
	}
	return nil
}

//UpdateUser 更新用户信息
func (c *AppConnector) UpdateUser(p *UserInfo) error {
	err := c.execSQL(updateUserSQL, p.Username, p.Password, p.GroupID, p.id)
	if err != nil {
		Error.Println(err)
		return err
	}
	return nil
}

//DeleteDomain 删除一条域数据
func (c *AppConnector) DeleteDomain(domainID int) error {
	err := c.execSQL(deleteDomainSQL, domainID)
	if err != nil {
		Error.Println(err)
		return err
	}
	return nil
}

//DeleteGroup 删除一条组数据
func (c *AppConnector) DeleteGroup(groupID int) error {
	err := c.execSQL(deleteGroupSQL, groupID)
	if err != nil {
		Error.Println(err)
		return err
	}
	return nil
}

//DeleteUser 删除一条用户数据
func (c *AppConnector) DeleteUser(userID int) error {
	err := c.execSQL(deleteUserSQL, userID)
	if err != nil {
		Error.Println(err)
		return err
	}
	return nil
}
