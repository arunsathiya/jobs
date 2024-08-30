package boards

import "encore/pkg/types"

func GetBoards() map[string]types.BoardFunc {
	return map[string]types.BoardFunc{
		"ashby":      Ashby,
		"bamboohr":   BambooHR,
		"greenhouse": Greenhouse,
		"lever":      Lever,
		"workable":   Workable,
	}
}
