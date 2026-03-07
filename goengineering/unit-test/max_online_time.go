package unittest

import "sort"

type userOnLine struct {
	uid        int64
	loginTime  int64
	logoutTime int64
}

// event 表示一个事件：时间和 delta（+1 表示登录，-1 表示登出）
type event struct {
	t     int64
	delta int
}

/*
统计活动在线人数在一天的什么时候达到峰值，如果有多个时刻达到峰值，输出第一个.
这里用户的登录和登出时间都是秒(相对于当天0时0分0秒过去了多少秒，在0-3600x24之间)
统计在哪个时刻(相对于当天0时0分0秒过去了多少秒)在线人数达到了峰值。
现在你已经有了这一天用户的登录和登出时间全量数据，设计一个算法来实现这个需求，
要求尽可能的高效。
*/

func getMaxOnlineTime(users []*userOnLine) int64 {
	n := len(users)
	// 构建事件数组
	events := make([]event, 0, 2*n)
	for i := 0; i < n; i++ {
		events = append(events, event{
			t:     users[i].loginTime,
			delta: 1,
		})
		events = append(events, event{
			t:     users[i].logoutTime,
			delta: -1,
		})
	}
	// 对事件按时间升序排序，如果时间相同，则登录事件（+1）排前面
	sort.Slice(events, func(i, j int) bool {
		if events[i].t == events[j].t {
			return events[i].delta > events[j].delta
		}
		return events[i].t < events[j].t
	})
	// 初始化当前在线人数
	currentOnline := 0
	// 初始化历史最大在线人数
	maxOnline := 0
	// 初始化峰值时间
	var peakTime int64
	i := 0
	for i < n*2 {
		// 为了处理多个事件在同一时刻发生，采用按时间分组累计 delta 的方法
		currentTime := events[i].t
		deltaSum := 0
		// 将所有在 currentTime 时刻的事件累加起来
		for i < n*2 && events[i].t == currentTime {
			deltaSum += events[i].delta
			i++
		}
		// 更新当前在线人数
		currentOnline += deltaSum
		// 只有当当前在线人数严格大于历史最大在线人数时，更新最大在线人数和对应的峰值时间
		if currentOnline > maxOnline {
			maxOnline = currentOnline
			peakTime = currentTime
		}
	}
	return peakTime
}
