package cert

func removeDup(items []string) []string {
	var ret []string
	existMap := make(map[string]bool)
	for _, v := range items {
		if !existMap[v] {
			ret = append(ret, v)
			existMap[v] = true
		}
	}
	return ret
}
