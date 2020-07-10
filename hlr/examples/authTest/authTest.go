package main

import "fmt"

func authInfoMarshal(domain string, username string, password string) string {
	return fmt.Sprintf(`
<document type="freeswitch/xml">
  <section name="directory">
    <domain name=%s>
      <params>
        <param name="dial-string" value="{^^:sip_invite_domain=%s:presence_id=%s@%s}${sofia_contact(*/%s@%s)},${verto_contact(%s@%s)}"/>
        <param name="jsonrpc-allowed-methods" value="verto"/>
        <param name="jsonrpc-allowed-event-channels" value="demo,conference,presence"/>
      </params>
      <variables>
        <variable name="record_stereo" value="true"/>
        <variable name="default_gateway" value="%s"/>
        <variable name="default_areacode" value="%s"/>
        <variable name="transfer_fallback_usernamesion" value="operator"/>
      </variables>
      <groups>
        <group name="default">
          <users>
            <user id="%s">
              <params>
                <param name="password" value="%s"/>
                <param name="vm-password" value="%s"/>
              </params>
              <variables>
                <variable name="toll_allow" value="domestic,international,local"/>
                <variable name="accountcode" value="%s"/>
                <variable name="user_context" value="default"/>
                <variable name="effective_caller_id_name" value="Extension %s"/>
                <variable name="effective_caller_id_number" value="%s"/>
                <variable name="outbound_caller_id_name" value="FS Conference"/>
                <variable name="outbound_caller_id_number" value="8888"/>
                <variable name="callgroup" value="techsupport"/>
              </variables>
            </user>  
          </users>
        </group>
      </groups>
    </domain>
  </section> 
</document>`,
		domain,
		domain, username, domain, username, domain, username, domain,
		domain,
		domain,
		username,
		password,
		username,
		username,
		username,
		username)
}

func main() {
	fmt.Println(authInfoMarshal("cbz", "jinzhuwei", "123qwe"))
}
