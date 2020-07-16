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

// 查找domain的内存位置,加读锁
// 注意，使用完需释放锁
func findDomainAndRLock(realm string) *DomainManage {
	amsDataManage.RLock()
	defer amsDataManage.RUnlock()
	thisDomain := amsDataManage.mapping[realm]
	if thisDomain == nil {
		return thisDomain
	}
	thisDomain.RLock()
	return thisDomain
}

// 定位domain的内存位置，加写锁
// 注意，使用完需释放锁
func locateDomainAndLock(realm string) *DomainManage {
	amsDataManage.RLock()
	defer amsDataManage.RUnlock()
	thisDomain := amsDataManage.mapping[realm]
	if thisDomain == nil {
		return thisDomain
	}
	thisDomain.Lock()
	return thisDomain
}

//加载归属于指定domain下users的信息
func reloadDomain(domain *DomainManage) error {
	// defer domain.Unlock()
	// domain.Lock()
	c, err := GetDBConnector()
	if err != nil || c == nil {
		return errors.New("get database connector failed")
	}
	queryStr := fmt.Sprintf(selectEnabledUsersSQL, domain.id)
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
			continue
		}
		user := domain.mapping[p.Username]
		//user不存在则新建，存在则修改
		if user == nil {
			user = new(UserCompleteInfo)
			user.Mutex = new(sync.Mutex)
			domain.mapping[p.Username] = user
			//计数器
			domain.agentCount++
			//初始化坐席状态信息
			user.Status = StatusLoggedOut
			user.State = StateIdle
			user.AnsweredCalls = 0
			user.TalkedTime = 0
			user.Talking = false
		}
		user.UserInfo = p
	}
	return nil
}

//ReloadAllData 从数据库加载所有数据，包括域、组、用户
func ReloadAllData() error {
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
			continue
		}
		//domain不存在则新建，存在则修改
		thisDomain := amsDataManage.mapping[p.Name]
		if thisDomain == nil {
			thisDomain = new(DomainManage)
			thisDomain.RWMutex = new(sync.RWMutex)
			thisDomain.mapping = make(map[string]*UserCompleteInfo)
			amsDataManage.mapping[p.Name] = thisDomain
		}
		thisDomain.DomainInfo = p

		thisDomain.Lock()
		reloadDomain(thisDomain)
		thisDomain.Unlock()
	}
	return nil
}

//注册状态变更
func userStatusSet(username string, realm string, status userstatusType) error {
	// amsDataManage.RLock()
	// domain := amsDataManage.mapping[realm]
	// if domain == nil {
	// 	amsDataManage.RUnlock()
	// 	return
	// }
	// domain.RLock()
	// amsDataManage.RUnlock()
	domain := findDomainAndRLock(realm)
	if domain == nil {
		return errors.New("The domain not exists")
	}
	//查询用户信息
	u := domain.mapping[username]
	if u == nil {
		domain.RUnlock()
		return errors.New("The user not exists")
	}
	u.Lock()
	if u.Status != status {
		Info.Printf("%s Status: from %s change to %s", username, u.Status, status)
		u.Status = status
	} else {
		Warning.Printf("%s Status is already %s", username, u.Status)
	}
	u.Unlock()
	domain.RUnlock()

	return nil
}

//坐席状态变更
func userStateSet(username string, realm string, state agentStateType) error {
	//查询域信息
	// amsDataManage.RLock()
	// domain := amsDataManage.mapping[realm]
	// if domain == nil {
	// 	amsDataManage.RUnlock()
	// 	return
	// }
	// domain.RLock()
	// amsDataManage.RUnlock()

	domain := findDomainAndRLock(realm)
	if domain == nil {
		return errors.New("The domain not exists")
	}
	//查询用户信息
	u := domain.mapping[username]
	if u == nil {
		domain.RUnlock()
		return errors.New("The user not exists")
	}
	u.Lock()
	if u.Status == StatusLoggedOut {
		u.Unlock()
		domain.RUnlock()
		return errors.New("The user status is logged out")
	}
	if u.State != state {
		Info.Printf("%s state: from %s change to %s", username, u.Status, state)
		u.State = state
	} else {
		Warning.Printf("%s state is already %s", username, u.State)
	}
	u.Unlock()
	domain.RUnlock()
	return nil
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

func deleteDomainData(realm string) error {

	amsDataManage.Lock()
	defer amsDataManage.Unlock()
	domain := amsDataManage.mapping[realm]
	if domain == nil {
		return errors.New("domain not found")
	}
	amsDataManage.mapping[realm] = nil
	return nil
}

// //用domainID查询domain信息
// func domainDataGetByID(domainID int) {

// }

// //用userID查询user信息
// func userDataGetByID(userID int) u *UserCompleteInfo {

// 	return u
// }
