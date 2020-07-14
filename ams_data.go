package ams

import (
	"errors"
	"fmt"
	"sync"
)

//DomainsManage ams全局数据管理，从数据库读取保存在内存中
var amsDataManage = struct {
	*sync.RWMutex
	mapping map[string]*DomainManage
}{
	RWMutex: new(sync.RWMutex),
	mapping: make(map[string]*DomainManage),
}

//加载特定域下的所有信息，域信息、所属域的所有用户信息
func loadDomainsFromDB(domain *DomainManage, domainID int) error {
	// defer domain.Unlock()
	// domain.Lock()
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
	defer amsDataManage.Unlock()
	amsDataManage.Lock()
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
		amsDataManage.mapping[p.Name] = thisDomain
		loadDomainsFromDB(thisDomain, p.id)
	}
	return nil
}

//注册状态变更
func userStatusSet(username string, realm string, status userstatusType) {
	//查询域信息
	amsDataManage.RLock()
	domain := amsDataManage.mapping[realm]
	if domain == nil {
		amsDataManage.RUnlock()
		return
	}
	domain.RLock()
	amsDataManage.RUnlock()
	//查询用户信息
	u := domain.mapping[username]
	if u == nil {
		domain.RUnlock()
		return
	}
	u.Lock()
	if u.Status != status {
		Info.Printf("%s Status: from %s change to %s", username, u.Status, status)
		u.Status = status
	} else {
		Warning.Printf("%s Status is alreay %s", username, u.Status)
	}
	u.Unlock()
	domain.RUnlock()
}

//坐席状态变更
func userStateSet(username string, realm string, state agentStateType) {
	//查询域信息
	amsDataManage.RLock()
	domain := amsDataManage.mapping[realm]
	if domain == nil {
		amsDataManage.RUnlock()
		return
	}
	domain.RLock()
	amsDataManage.RUnlock()
	//查询用户信息
	u := domain.mapping[username]
	if u == nil {
		domain.RUnlock()
		return
	}
	u.Lock()
	if u.State != state {
		Info.Printf("%s state: from %s change to %s", username, u.Status, state)
		u.State = state
	} else {
		Warning.Printf("%s state is alreay %s", username, u.State)
	}
	u.Unlock()
	domain.RUnlock()
}

//通话状态设置
func userTalkingSet(username string, realm string, talking bool) {
	//查询域信息
	amsDataManage.RLock()
	domain := amsDataManage.mapping[realm]
	if domain == nil {
		amsDataManage.RUnlock()
		return
	}
	domain.RLock()
	amsDataManage.RUnlock()
	//查询用户信息
	u := domain.mapping[username]
	if u == nil {
		domain.RUnlock()
		return
	}
	u.Lock()
	if u.Talking != talking {
		Info.Printf("%s talking from %t to %t", username, u.Talking, talking)
		u.Talking = talking
	} else {
		Warning.Printf("%s talking is alreay %t", username, u.Talking)
	}
	u.Unlock()
	domain.RUnlock()
}
