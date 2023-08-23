// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"
)

type Animal interface {
	IsAnimal()
	GetClass() Class
	GetGender() Gender
}

type Pet interface {
	IsAnimal()
	IsPet()
	GetClass() Class
	GetGender() Gender
	GetName() string
}

type Alligator struct {
	Class     Class  `json:"class"`
	Gender    Gender `json:"gender"`
	Name      string `json:"name"`
	Dangerous string `json:"dangerous"`
}

func (Alligator) IsPet()                 {}
func (this Alligator) GetClass() Class   { return this.Class }
func (this Alligator) GetGender() Gender { return this.Gender }
func (this Alligator) GetName() string   { return this.Name }

func (Alligator) IsAnimal() {}

type Cat struct {
	Class  Class   `json:"class"`
	Gender Gender  `json:"gender"`
	Name   string  `json:"name"`
	Type   CatType `json:"type"`
}

func (Cat) IsPet()                 {}
func (this Cat) GetClass() Class   { return this.Class }
func (this Cat) GetGender() Gender { return this.Gender }
func (this Cat) GetName() string   { return this.Name }

func (Cat) IsAnimal() {}

type Details struct {
	Forename string `json:"forename"`
	Surname  string `json:"surname"`
}

type Dog struct {
	Breed  DogBreed `json:"breed"`
	Class  Class    `json:"class"`
	Gender Gender   `json:"gender"`
	Name   string   `json:"name"`
}

func (Dog) IsPet()                 {}
func (this Dog) GetClass() Class   { return this.Class }
func (this Dog) GetGender() Gender { return this.Gender }
func (this Dog) GetName() string   { return this.Name }

func (Dog) IsAnimal() {}

type Employee struct {
	Details       *Details       `json:"details,omitempty"`
	ID            int            `json:"id"`
	HasChildren   bool           `json:"hasChildren"`
	MaritalStatus *MaritalStatus `json:"maritalStatus,omitempty"`
	Nationality   Nationality    `json:"nationality"`
	Pets          []Pet          `json:"pets,omitempty"`
}

func (Employee) IsEntity() {}

type Mouse struct {
	Class  Class  `json:"class"`
	Gender Gender `json:"gender"`
	Name   string `json:"name"`
}

func (Mouse) IsPet()                 {}
func (this Mouse) GetClass() Class   { return this.Class }
func (this Mouse) GetGender() Gender { return this.Gender }
func (this Mouse) GetName() string   { return this.Name }

func (Mouse) IsAnimal() {}

type Pony struct {
	Class  Class  `json:"class"`
	Gender Gender `json:"gender"`
	Name   string `json:"name"`
}

func (Pony) IsPet()                 {}
func (this Pony) GetClass() Class   { return this.Class }
func (this Pony) GetGender() Gender { return this.Gender }
func (this Pony) GetName() string   { return this.Name }

func (Pony) IsAnimal() {}

type CatType string

const (
	CatTypeHome   CatType = "HOME"
	CatTypeStreet CatType = "STREET"
)

var AllCatType = []CatType{
	CatTypeHome,
	CatTypeStreet,
}

func (e CatType) IsValid() bool {
	switch e {
	case CatTypeHome, CatTypeStreet:
		return true
	}
	return false
}

func (e CatType) String() string {
	return string(e)
}

func (e *CatType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = CatType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid CatType", str)
	}
	return nil
}

func (e CatType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type Class string

const (
	ClassFish    Class = "Fish"
	ClassMammal  Class = "Mammal"
	ClassReptile Class = "Reptile"
)

var AllClass = []Class{
	ClassFish,
	ClassMammal,
	ClassReptile,
}

func (e Class) IsValid() bool {
	switch e {
	case ClassFish, ClassMammal, ClassReptile:
		return true
	}
	return false
}

func (e Class) String() string {
	return string(e)
}

func (e *Class) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = Class(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid Class", str)
	}
	return nil
}

func (e Class) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type DogBreed string

const (
	DogBreedGoldenRetriever  DogBreed = "GOLDEN_RETRIEVER"
	DogBreedPoodle           DogBreed = "POODLE"
	DogBreedRottweiler       DogBreed = "ROTTWEILER"
	DogBreedYorkshireTerrier DogBreed = "YORKSHIRE_TERRIER"
)

var AllDogBreed = []DogBreed{
	DogBreedGoldenRetriever,
	DogBreedPoodle,
	DogBreedRottweiler,
	DogBreedYorkshireTerrier,
}

func (e DogBreed) IsValid() bool {
	switch e {
	case DogBreedGoldenRetriever, DogBreedPoodle, DogBreedRottweiler, DogBreedYorkshireTerrier:
		return true
	}
	return false
}

func (e DogBreed) String() string {
	return string(e)
}

func (e *DogBreed) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = DogBreed(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid DogBreed", str)
	}
	return nil
}

func (e DogBreed) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type Gender string

const (
	GenderFemale  Gender = "FEMALE"
	GenderMale    Gender = "MALE"
	GenderUnknown Gender = "UNKNOWN"
)

var AllGender = []Gender{
	GenderFemale,
	GenderMale,
	GenderUnknown,
}

func (e Gender) IsValid() bool {
	switch e {
	case GenderFemale, GenderMale, GenderUnknown:
		return true
	}
	return false
}

func (e Gender) String() string {
	return string(e)
}

func (e *Gender) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = Gender(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid Gender", str)
	}
	return nil
}

func (e Gender) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type MaritalStatus string

const (
	MaritalStatusEngaged MaritalStatus = "ENGAGED"
	MaritalStatusMarried MaritalStatus = "MARRIED"
)

var AllMaritalStatus = []MaritalStatus{
	MaritalStatusEngaged,
	MaritalStatusMarried,
}

func (e MaritalStatus) IsValid() bool {
	switch e {
	case MaritalStatusEngaged, MaritalStatusMarried:
		return true
	}
	return false
}

func (e MaritalStatus) String() string {
	return string(e)
}

func (e *MaritalStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = MaritalStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid MaritalStatus", str)
	}
	return nil
}

func (e MaritalStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type Nationality string

const (
	NationalityAmerican  Nationality = "AMERICAN"
	NationalityDutch     Nationality = "DUTCH"
	NationalityEnglish   Nationality = "ENGLISH"
	NationalityGerman    Nationality = "GERMAN"
	NationalityIndian    Nationality = "INDIAN"
	NationalitySpanish   Nationality = "SPANISH"
	NationalityUkrainian Nationality = "UKRAINIAN"
)

var AllNationality = []Nationality{
	NationalityAmerican,
	NationalityDutch,
	NationalityEnglish,
	NationalityGerman,
	NationalityIndian,
	NationalitySpanish,
	NationalityUkrainian,
}

func (e Nationality) IsValid() bool {
	switch e {
	case NationalityAmerican, NationalityDutch, NationalityEnglish, NationalityGerman, NationalityIndian, NationalitySpanish, NationalityUkrainian:
		return true
	}
	return false
}

func (e Nationality) String() string {
	return string(e)
}

func (e *Nationality) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = Nationality(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid Nationality", str)
	}
	return nil
}

func (e Nationality) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
