package main

const MinChips int = 1
const StartingChips int = 50

type Player struct {
	ID       int
	Username string
	Chips    int
	Wins     int
	Ties     int
	Losses   int
}
