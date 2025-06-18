package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIPPool(t *testing.T) {
	t.Run("should create new IP pool with valid CIDR", func(t *testing.T) {
		pool, err := NewIPPool("10.0.0.0/24")
		require.NoError(t, err)
		assert.NotNil(t, pool)
		assert.Equal(t, "10.0.0.0/24", pool.network)
		assert.Equal(t, "10.0.0.1", pool.serverIP)
	})

	t.Run("should fail with invalid CIDR", func(t *testing.T) {
		_, err := NewIPPool("invalid-cidr")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid CIDR")
	})

	t.Run("should fail with IPv6", func(t *testing.T) {
		_, err := NewIPPool("2001:db8::/32")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IPv6 not supported")
	})

	t.Run("should fail with single host", func(t *testing.T) {
		_, err := NewIPPool("10.0.0.1/32")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network too small")
	})
}

func TestIPPool_AllocateIP(t *testing.T) {
	pool, err := NewIPPool("10.0.0.0/29") // 8 addresses total
	require.NoError(t, err)

	t.Run("should allocate first available IP", func(t *testing.T) {
		ip, err := pool.AllocateIP()
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.2", ip) // .1 is server, .2 is first client
	})

	t.Run("should allocate sequential IPs", func(t *testing.T) {
		ip1, err := pool.AllocateIP()
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.3", ip1)

		ip2, err := pool.AllocateIP()
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.4", ip2)
	})

	t.Run("should fail when pool is exhausted", func(t *testing.T) {
		// Allocate remaining IPs (.5, .6 - .7 is broadcast)
		_, err := pool.AllocateIP()
		require.NoError(t, err) // .5
		_, err = pool.AllocateIP()
		require.NoError(t, err) // .6

		// Now pool should be exhausted
		_, err = pool.AllocateIP()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no available IP addresses")
	})
}

func TestIPPool_AllocateSpecificIP(t *testing.T) {
	pool, err := NewIPPool("10.0.0.0/28") // 16 addresses
	require.NoError(t, err)

	t.Run("should allocate specific IP", func(t *testing.T) {
		err := pool.AllocateSpecificIP("10.0.0.5")
		require.NoError(t, err)
		assert.True(t, pool.IsAllocated("10.0.0.5"))
	})

	t.Run("should fail to allocate already allocated IP", func(t *testing.T) {
		err := pool.AllocateSpecificIP("10.0.0.5")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IP address already allocated")
	})

	t.Run("should fail to allocate server IP", func(t *testing.T) {
		err := pool.AllocateSpecificIP("10.0.0.1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reserved for server")
	})

	t.Run("should fail to allocate IP outside network", func(t *testing.T) {
		err := pool.AllocateSpecificIP("192.168.1.1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IP address not in network range")
	})

	t.Run("should fail to allocate network address", func(t *testing.T) {
		err := pool.AllocateSpecificIP("10.0.0.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network address")
	})

	t.Run("should fail to allocate broadcast address", func(t *testing.T) {
		err := pool.AllocateSpecificIP("10.0.0.15")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "broadcast address")
	})
}

func TestIPPool_ReleaseIP(t *testing.T) {
	pool, err := NewIPPool("10.0.0.0/28")
	require.NoError(t, err)

	t.Run("should release allocated IP", func(t *testing.T) {
		ip, err := pool.AllocateIP()
		require.NoError(t, err)

		err = pool.ReleaseIP(ip)
		require.NoError(t, err)
		assert.False(t, pool.IsAllocated(ip))
	})

	t.Run("should fail to release non-allocated IP", func(t *testing.T) {
		err := pool.ReleaseIP("10.0.0.10")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IP address not allocated")
	})

	t.Run("should fail to release IP outside network", func(t *testing.T) {
		err := pool.ReleaseIP("192.168.1.1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IP address not in network range")
	})
}

func TestIPPool_IsAllocated(t *testing.T) {
	pool, err := NewIPPool("10.0.0.0/28")
	require.NoError(t, err)

	t.Run("should return false for non-allocated IP", func(t *testing.T) {
		assert.False(t, pool.IsAllocated("10.0.0.5"))
	})

	t.Run("should return true for allocated IP", func(t *testing.T) {
		ip, err := pool.AllocateIP()
		require.NoError(t, err)
		assert.True(t, pool.IsAllocated(ip))
	})

	t.Run("should return true for server IP", func(t *testing.T) {
		assert.True(t, pool.IsAllocated("10.0.0.1"))
	})
}

func TestIPPool_GetServerIP(t *testing.T) {
	pool, err := NewIPPool("192.168.100.0/24")
	require.NoError(t, err)

	t.Run("should return server IP", func(t *testing.T) {
		serverIP := pool.GetServerIP()
		assert.Equal(t, "192.168.100.1", serverIP)
	})
}

func TestIPPool_GetAllocatedIPs(t *testing.T) {
	pool, err := NewIPPool("10.0.0.0/28")
	require.NoError(t, err)

	t.Run("should return empty list initially", func(t *testing.T) {
		allocated := pool.GetAllocatedIPs()
		assert.Empty(t, allocated)
	})

	t.Run("should return allocated IPs", func(t *testing.T) {
		ip1, err := pool.AllocateIP()
		require.NoError(t, err)
		ip2, err := pool.AllocateIP()
		require.NoError(t, err)

		allocated := pool.GetAllocatedIPs()
		assert.Len(t, allocated, 2)
		assert.Contains(t, allocated, ip1)
		assert.Contains(t, allocated, ip2)
	})
}

func TestIPPool_GetAvailableCount(t *testing.T) {
	pool, err := NewIPPool("10.0.0.0/29") // 8 total, -2 (network, broadcast), -1 (server) = 5 available
	require.NoError(t, err)

	t.Run("should return correct available count", func(t *testing.T) {
		count := pool.GetAvailableCount()
		assert.Equal(t, 5, count)
	})

	t.Run("should decrease after allocation", func(t *testing.T) {
		_, err := pool.AllocateIP()
		require.NoError(t, err)

		count := pool.GetAvailableCount()
		assert.Equal(t, 4, count)
	})

	t.Run("should increase after release", func(t *testing.T) {
		ip, err := pool.AllocateIP()
		require.NoError(t, err)

		err = pool.ReleaseIP(ip)
		require.NoError(t, err)

		count := pool.GetAvailableCount()
		assert.Equal(t, 4, count) // Back to 4 (one still allocated from previous test)
	})
}

func TestIPPool_GetNetworkInfo(t *testing.T) {
	pool, err := NewIPPool("172.16.0.0/16")
	require.NoError(t, err)

	t.Run("should return network info", func(t *testing.T) {
		info := pool.GetNetworkInfo()
		assert.Equal(t, "172.16.0.0/16", info.Network)
		assert.Equal(t, "172.16.0.1", info.ServerIP)
		assert.Equal(t, "172.16.0.0", info.NetworkAddress)
		assert.Equal(t, "172.16.255.255", info.BroadcastAddress)
		assert.Equal(t, 65534, info.TotalHosts) // 2^16 - 2
	})
}