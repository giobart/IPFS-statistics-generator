package lib

import (
	"github.com/ip2location/ip2location-go"
	ma "github.com/multiformats/go-multiaddr"
)

type PeerGeolocation struct {
	ip42locdb *ip2location.DB
	ip62locdb *ip2location.DB
}

/* Given a peer  -> set to the Peer the City and the Country from his ip address */
func (p *PeerGeolocation) SetPeerCity(peer *Peer) {

	var locdb *ip2location.DB
	ipAddr := ""

	// construct multiaddr from a string (err signals parse failure)
	multiaddr, err := ma.NewMultiaddr(peer.Addr)
	if err != nil {
		return
	}

	//swapping db between ipv6 or ipv4 to perform geolocation query
	for _, v := range multiaddr.Protocols() {
		if v.Code == ma.P_IP4 {
			locdb = p.ip42locdb
			ipAddr, _ = multiaddr.ValueForProtocol(v.Code)
			continue
		}
		if v.Code == ma.P_IP6 {
			locdb = p.ip62locdb
			ipAddr, _ = multiaddr.ValueForProtocol(v.Code)
			continue
		}

	}

	//fetching country and city from ipv4/ipv6 address db
	if locdb != nil {
		results, err := locdb.Get_all(ipAddr)
		if err != nil {
			log.Error(err)
		} else {
			if results.Country_long != "-" {
				peer.Nation = results.Country_long
				peer.City = results.City
				peer.Lat = results.Latitude
				peer.Lon = results.Longitude
			}
		}
	}

}

func (p *PeerGeolocation) Init(ip4dbloc string, ip6dbloc string) error {

	ip42locdb, err := ip2location.OpenDB(ip4dbloc)
	if err != nil {
		return err
	}
	p.ip42locdb = ip42locdb

	ip62locdb, err := ip2location.OpenDB(ip6dbloc)
	if err != nil {
		return err
	}
	p.ip62locdb = ip62locdb

	return nil

}
