package tools

import "myapp/models"

type fakeService struct {
	repo *workDatabase
}

type workDatabase struct {
	workByID map[uint]models.Work
}

func (wd *workDatabase) GetWorkByID(id uint) (models.Work, error) {
}
