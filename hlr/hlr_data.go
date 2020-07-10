package hlr

import (
	"errors"
	"fmt"
	"sync"
)

//DomainsManage HLR全局数据管理，从数据库读取保存在内存中
var hlrDataManage = make(map[string]*DomainManage)

//加载特定域下的所有信息，域信息、所属域的所有用户信息
func loadDomainsFromDB(domain *DomainManage, domainID int) error {
	defer domain.Unlock()
	domain.Lock()
	c, err := GetDBConnector()
	if err != nil || c == nil {
		return errors.New("get database connector failed")
	}
	queryStr := fmt.Sprintf(selectEnabledUsersSQL, domainID)
	rows, err := c.dbConn.Query(queryStr)
	if err != nil {
		Error.Println("load domains from database ", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		p := new(UserInfo)
		err := rows.Scan(&p.id, &p.Username, &p.Password, &p.GroupID)
		if err != nil {
			Error.Println(err)
		}
		user := new(UserCompleteInfo)
		user.Mutex = new(sync.Mutex)
		user.UserInfo = p
		user.Status = StatusLoggedOut
		user.State = StateIdle
		user.AnsweredCalls = 0
		user.TalkedTime = 0
		user.Talking = false
		domain.mapping[p.Username] = user
	}
	return nil
}

//LoadAllDataFromDB 从数据库加载所有数据，包括域、组、用户
func LoadAllDataFromDB() error {
	c, err := GetDBConnector()
	if err != nil || c == nil {
		return errors.New("get DB connector failed")
	}
	rows, err := c.dbConn.Query(selectEnabledDomainsSQL)
	if err != nil {
		Error.Println(err.Error())
		return err
	}
	defer rows.Close()
	for rows.Next() {
		p := new(DomainInfo)
		err := rows.Scan(&p.id, &p.Name, &p.TenantID, &p.Company)
		if err != nil {
			Error.Println(err)
		}
		thisDomain := new(DomainManage)
		thisDomain.RWMutex = new(sync.RWMutex)
		thisDomain.DomainInfo = p
		thisDomain.mapping = make(map[string]*UserCompleteInfo)
		hlrDataManage[p.Name] = thisDomain
		loadDomainsFromDB(thisDomain, p.id)
	}
	return nil
}
