package hlr

import (
	"errors"
	"fmt"
	"sync"
)

//DomainsManage hlr全局数据管理，从数据库读取保存在内存中
var hlrDataManage = struct {
	*sync.RWMutex
	mapping map[string]*DomainManage
}{
	RWMutex: new(sync.RWMutex),
	mapping: make(map[string]*DomainManage),
}

// 查找domain的内存位置,读锁
// 注意，使用完需释放锁
func findDomainAndRLock(realm string) *DomainManage {
	hlrDataManage.RLock()
	defer hlrDataManage.RUnlock()
	thisDomain := hlrDataManage.mapping[realm]
	if thisDomain == nil {
		return thisDomain
	}
	thisDomain.RLock()
	return thisDomain
}

// 定位domain的内存位置，写锁
// 注意，使用完需释放锁
func locateDomainAndLock(realm string) *DomainManage {
	hlrDataManage.RLock()
	defer hlrDataManage.RUnlock()
	thisDomain := hlrDataManage.mapping[realm]
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
	defer hlrDataManage.Unlock()
	hlrDataManage.Lock()
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
		thisDomain := hlrDataManage.mapping[p.Name]
		if thisDomain == nil {
			thisDomain = new(DomainManage)
			thisDomain.RWMutex = new(sync.RWMutex)
			thisDomain.mapping = make(map[string]*UserCompleteInfo)
			hlrDataManage.mapping[p.Name] = thisDomain
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
	// hlrDataManage.RLock()
	// domain := hlrDataManage.mapping[realm]
	// if domain == nil {
	// 	hlrDataManage.RUnlock()
	// 	return
	// }
	// domain.RLock()
	// hlrDataManage.RUnlock()
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
	// hlrDataManage.RLock()
	// domain := hlrDataManage.mapping[realm]
	// if domain == nil {
	// 	hlrDataManage.RUnlock()
	// 	return
	// }
	// domain.RLock()
	// hlrDataManage.RUnlock()

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
	hlrDataManage.RLock()
	domain := hlrDataManage.mapping[realm]
	if domain == nil {
		hlrDataManage.RUnlock()
		return
	}
	domain.RLock()
	hlrDataManage.RUnlock()
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

func addDomainData(d *DomainInfo) error {
	hlrDataManage.Lock()
	defer hlrDataManage.Unlock()
	thisDomain := hlrDataManage.mapping[d.Name]
	if thisDomain != nil {
		return errors.New("domain is already exists in data")
	}
	thisDomain = new(DomainManage)

	thisDomain.RWMutex = new(sync.RWMutex)
	thisDomain.mapping = make(map[string]*UserCompleteInfo)
	hlrDataManage.mapping[d.Name] = thisDomain
	thisDomain.DomainInfo = d

	return nil
}

func deleteDomainData(realm string) error {

	hlrDataManage.Lock()
	defer hlrDataManage.Unlock()
	domain := hlrDataManage.mapping[realm]
	if domain == nil {
		return errors.New("domain not found")
	}
	hlrDataManage.mapping[realm] = nil
	return nil
}

func deleteUserData(username string, realm string) error {
	hlrDataManage.RLock()
	domain := hlrDataManage.mapping[realm]
	domain.Lock()
	hlrDataManage.RUnlock()

	domain.mapping[username] = nil
	domain.Unlock()

	return nil
}

func addUserData(userInfo UserInfo, realm string) error {
	hlrDataManage.RLock()
	domain := hlrDataManage.mapping[realm]
	domain.Lock()
	hlrDataManage.RUnlock()
	user := domain.mapping[userInfo.Username]
	if user != nil {
		Error.Println("user", userInfo.Username, "already exists")
		domain.Unlock()
		return errors.New("user already exists")
	}

	user = new(UserCompleteInfo)
	user.Mutex = new(sync.Mutex)
	domain.mapping[userInfo.Username] = user
	//计数器
	domain.agentCount++
	//初始化坐席状态信息
	user.Status = StatusLoggedOut
	user.State = StateIdle
	user.AnsweredCalls = 0
	user.TalkedTime = 0
	user.Talking = false

	domain.Unlock()
	return nil
}
