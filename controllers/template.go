package controllers

type RoutingData struct {
	UDPPort          uint16
	TCPPort          uint16
	AdvertiseAddress string
	Generation       int
	Rules            []RoutingRule
}

type RoutingRule struct {
	Domain     string
	Headnumber string
	Owner      string
	Backend    string
}
