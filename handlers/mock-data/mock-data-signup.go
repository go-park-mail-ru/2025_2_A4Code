package mock_data

type MockDataSignup []map[string]string

func NewData() MockDataSignup {

	users := []map[string]string{
		{"login": "admin", "password": "kwdhcslkanwddj", "username": "Admin User", "dateofbirth": "12/09/2005", "gender": "male"},
		{"login": "support", "password": "dkchcslkanwddj", "username": "Support Team", "dateofbirth": "05/09/2003", "gender": "female"},
		{"login": "news", "password": "dfldhcslkanwddj", "username": "Newsletter", "dateofbirth": "03/08/2005", "gender": "male"},
		{"login": "noreply", "password": "mndkfnhcslkanwddj", "username": "Service Bot", "dateofbirth": "08/10/2022", "gender": "male"},
		{"login": "boss", "password": "kdkcslkanwddj", "username": "The Boss", "dateofbirth": "01/01/2023", "gender": "female"},
	}

	return users
}
