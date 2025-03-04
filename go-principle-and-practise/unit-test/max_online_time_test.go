package unittest

import "testing"

// 测试用例结构
type testCase struct {
	name     string
	users    []*userOnLine
	expected int64
}

func TestGetMaxOnlineTime(t *testing.T) {
	testCases := []testCase{
		{
			name: "Basic case",
			users: []*userOnLine{
				{uid: 1, loginTime: 100, logoutTime: 300},
				{uid: 2, loginTime: 200, logoutTime: 400},
				{uid: 3, loginTime: 250, logoutTime: 350},
				{uid: 4, loginTime: 150, logoutTime: 280},
			},
			expected: 250,
		},
		{
			name: "All users log in at the same time",
			users: []*userOnLine{
				{uid: 1, loginTime: 100, logoutTime: 200},
				{uid: 2, loginTime: 100, logoutTime: 300},
				{uid: 3, loginTime: 100, logoutTime: 400},
			},
			expected: 100,
		},
		{
			name: "Users log in and out sequentially",
			users: []*userOnLine{
				{uid: 1, loginTime: 100, logoutTime: 150},
				{uid: 2, loginTime: 150, logoutTime: 200},
				{uid: 3, loginTime: 200, logoutTime: 250},
			},
			expected: 100, // 第一个用户登录时刻是最大在线人数首次出现的时刻
		},
		{
			name: "Only one user",
			users: []*userOnLine{
				{uid: 1, loginTime: 500, logoutTime: 800},
			},
			expected: 500,
		},
		{
			name: "Users log out at the same time",
			users: []*userOnLine{
				{uid: 1, loginTime: 100, logoutTime: 300},
				{uid: 2, loginTime: 200, logoutTime: 300},
				{uid: 3, loginTime: 250, logoutTime: 300},
			},
			expected: 250,
		},
		{
			name: "Users stay online the whole day",
			users: []*userOnLine{
				{uid: 1, loginTime: 0, logoutTime: 86400},
				{uid: 2, loginTime: 0, logoutTime: 86400},
				{uid: 3, loginTime: 0, logoutTime: 86400},
			},
			expected: 0,
		},
		{
			name: "Users log in at different times",
			users: []*userOnLine{
				{uid: 1, loginTime: 100, logoutTime: 500},
				{uid: 2, loginTime: 200, logoutTime: 600},
				{uid: 3, loginTime: 300, logoutTime: 700},
				{uid: 4, loginTime: 400, logoutTime: 800},
				{uid: 5, loginTime: 500, logoutTime: 900},
			},
			expected: 400,
		},
		{
			name:     "No users",
			users:    []*userOnLine{},
			expected: 0, // 没有用户，默认返回 0
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getMaxOnlineTime(tc.users)
			if result != tc.expected {
				t.Errorf("Expected %d, but got %d", tc.expected, result)
			}
		})
	}
}
