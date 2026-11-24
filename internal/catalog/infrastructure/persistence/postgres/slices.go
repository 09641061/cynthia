package postgres

func insertAt(values []string, value string, position int) []string {
	values = append(values, "")
	copy(values[position+1:], values[position:])
	values[position] = value
	return values
}

func removeID(values []string, id string) []string {
	for index, value := range values {
		if value == id {
			return append(values[:index], values[index+1:]...)
		}
	}
	return values
}

func moveWithin(values []string, id string, position int) []string {
	return insertAt(removeID(values, id), id, position)
}
