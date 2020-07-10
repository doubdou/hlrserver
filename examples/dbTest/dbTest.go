package main

import (
	"fmt"
	"hlr"

	_ "github.com/lib/pq"
)

func main() {
	hlr.LoadConfig()
	c, err := hlr.DBDriver()
	if err != nil {
		return
	}
	c.CreateTable()
	c.InsertDomain("ai-ym.com", 300012, "cbz", true)
	c.InsertGroup("dev", "开发中心", 0, 0)
	c.InsertUser("15606130692", "123qwe", 0)
	c.InsertUser("15371870149", "123qwe", 0)
	g, err := c.ReadGroup("1")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("read group info:", g.Name, g.GroupDesc, g.ParentID, g.DomainID)
	g.GroupDesc = "new desc开发中心"
	err = c.UpdateGroup(g)
	if err != nil {
		fmt.Println("update group fail ", err)
	}
	err = c.DeleteUser(2)
	if err != nil {
		fmt.Println("delete user fail ", err)
	}

	c.CloseDBDriver()
}
