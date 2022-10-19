package metric

func TruncateTags(maxLength int, tags map[string]string) map[string]string {
	for name := range tags {
		maxLen := maxLength - len(name) - len("=")
		if len(tags[name]) > maxLen {
			tags[name] = tags[name][:maxLen]
		}
	}
	return tags
}
