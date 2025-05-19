package fastdns

import "github.com/vishvananda/netlink"

// isActiveNeighbour checks if a neighbor is in an active state.
// It checks if the neighbor is in a valid state (e.g., reachable, permanent).
func isActiveNeighbour(neigh netlink.Neigh) bool {
	return (neigh.State & (netlink.NUD_PERMANENT | netlink.NUD_REACHABLE | netlink.NUD_STALE | netlink.NUD_PROBE | netlink.NUD_DELAY)) != 0
}
