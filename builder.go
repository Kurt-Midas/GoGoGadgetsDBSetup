package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/mgo.v2"
	"strings"
	// "gopkg.in/mgo.v2/bson"
	"log"
)

type About struct {
	Version string
}

func main() {
	// fmt.Println("Test only: this code grabs 'sqlite-latest.sqlite' from the working directory")
	//  "Found local copy of sqlite-latest.sqlite! Do you want to use this or retrieve a fresh copy?"

	db, err := sql.Open("sqlite3", "./sqlite-latest.sqlite")
	checkErr(err)
	defer db.Close()

	mSession, m_err := mgo.Dial("localhost:27017")
	checkErr(m_err)
	defer mSession.Close()

	setList := make([]map[int]bool, 0)

	scrapSet := scrapify(db, mSession)
	fmt.Println("Scrap set found", len(scrapSet), "unique values")
	setList = append(setList, scrapSet)

	planetSet := planetify(db, mSession)
	fmt.Println("Planet set found", len(planetSet), "unique values")
	setList = append(setList, planetSet)

	resourceSet := resourcify(mSession)
	fmt.Println("Resource set found", len(resourceSet), "unique values")
	setList = append(setList, resourceSet) //this should be redundant to planetSet

	allTypeIDs := setJoin(setList)
	fmt.Println("About to detalify", len(allTypeIDs), "unique values")

	detailify(db, mSession, allTypeIDs)

	fmt.Println("Success! Exiting.")
}

func setJoin(setList []map[int]bool) map[int]bool {
	allTypeIDs := make(map[int]bool)
	for x := range setList {
		for k, _ := range setList[x] {
			allTypeIDs[k] = true
		}
	}
	// fmt.Println("New size of allTypeIDs is", len(allTypeIDs))
	return allTypeIDs
}

/*
Detailify and its stuff
*/
func detailify(sql *sql.DB, mdb *mgo.Session, set map[int]bool) {

	if len(set) == 0 {
		fmt.Println("Length of detail set is 0. Something went wrong, panicking")
		panic("Set length is 0")
	}

	sliceset := make([]interface{}, 0)
	for key, _ := range set {
		sliceset = append(sliceset, key)
	}

	var start int = 0
	var end int = 999
	itemDetailsList := make([]itemdetails, 0)

	for start < len(set) {

		if end > len(set) {
			end = len(set) - 1
		}
		// fmt.Println("Querying between set start and end:", start, end)

		args := sliceset[start:end]
		query := strings.Join([]string{
			"SELECT t.typeID, t.typeName, t.marketGroupID, t.groupID, g.categoryID ",
			"FROM invTypes t INNER JOIN invGroups g ON t.groupID = g.groupID ",
			"AND t.typeID IN (?",
			strings.Repeat(",?", len(args)-1),
			")"}, "")
		stmt, err := sql.Prepare(query)
		checkErr(err)
		rows, err := stmt.Query(args...)
		checkErr(err)

		for rows.Next() {
			var typeID int
			var typeName string
			var marketGroupID int
			var groupID int
			var categoryID int
			rows.Scan(&typeID, &typeName, &marketGroupID, &groupID, &categoryID)
			// fmt.Println(typeID, typeName, marketGroupID, groupID, categoryID)
			itemDetailsList = append(itemDetailsList, itemdetails{typeID, typeName, marketGroupID, groupID, categoryID})
		}
		start += 1000
		end += 1000
	}

	detailsdb := mdb.DB("sde").C("itemdetails")
	detailsdb.RemoveAll(nil)
	for x := range itemDetailsList {
		// fmt.Println(insertableRecipes[x])
		m_err := detailsdb.Insert(&itemDetailsList[x])
		checkErr(m_err)
	}
}

type itemdetails struct {
	TypeID        int
	TypeName      string
	MarketGroupID int
	GroupID       int
	CategoryID    int
}

/*
Resourcify and its stuff
*/
type planetAndResources struct {
	PlanetID   int
	PlanetName string
	Resources  [5]int
}

func resourcify(mdb *mgo.Session) map[int]bool {
	// fmt.Println("Inside 'resourcify'")
	set := make(map[int]bool)
	//as far as I know there's no SDE for this one. I need to use static values. Forgive me Hao Chen! Actually nvm, fuck you.
	// planetIds := [8]int{11,12,13,2014,2015,2016,2017,2063}
	planetList := make([]planetAndResources, 0)
	planetList = append(planetList, planetAndResources{11, "Temperate", [5]int{2268, 2287, 2073, 2305, 2288}})
	planetList = append(planetList, planetAndResources{12, "Ice", [5]int{2268, 2272, 2073, 2286, 2310}})
	planetList = append(planetList, planetAndResources{13, "Gas", [5]int{2268, 2267, 2309, 2310, 2311}})
	planetList = append(planetList, planetAndResources{2014, "Oceanic", [5]int{2268, 2287, 2073, 2286, 2288}})
	planetList = append(planetList, planetAndResources{2015, "Lava", [5]int{2267, 2272, 2308, 2307, 2306}})
	planetList = append(planetList, planetAndResources{2016, "Barren", [5]int{2268, 2267, 2073, 2270, 2288}})
	planetList = append(planetList, planetAndResources{2017, "Storm", [5]int{2268, 2267, 2308, 2309, 2310}})
	planetList = append(planetList, planetAndResources{2063, "Plasma", [5]int{2267, 2272, 2270, 2308, 2306}})

	pidb := mdb.DB("sde").C("planetresources")
	pidb.RemoveAll(nil)
	for x := range planetList {
		// fmt.Println(insertableRecipes[x])
		m_err := pidb.Insert(&planetList[x])
		checkErr(m_err)
		for i := range planetList[x].Resources {
			set[planetList[x].Resources[i]] = true
		}
	}
	return set
}

/*
Planetify and its stuff
*/
type schematicDetails struct {
	TypeID   int
	Quantity int
	IsInput  int
}

type piRecipe struct {
	TypeID   int
	Quantity int
	Inputs   []recipeInput
}

type recipeInput struct {
	TypeID   int
	Quantity int
}

func planetify(sql *sql.DB, mdb *mgo.Session) map[int]bool {
	// fmt.Println("Inside 'planetify'")
	schematicLines, err := sql.Query("select * from planetschematicstypemap")
	defer schematicLines.Close()
	checkErr(err)

	set := make(map[int]bool)

	schematicMap := make(map[int][]schematicDetails)

	for schematicLines.Next() {
		var schematicId int
		var typeID int
		var quantity int
		var isInput int
		schematicLines.Scan(&schematicId, &typeID, &quantity, &isInput)

		if schematic, prs := schematicMap[schematicId]; prs {
			schematic = append(schematic, schematicDetails{typeID, quantity, isInput})
			schematicMap[schematicId] = schematic
		} else {
			schematic := make([]schematicDetails, 0)
			schematic = append(schematic, schematicDetails{typeID, quantity, isInput})
			schematicMap[schematicId] = schematic
		}
	}

	insertableRecipes := make([]piRecipe, 0)
	// var key int
	var value []schematicDetails
	for _, value = range schematicMap {
		var output recipeInput
		input := make([]recipeInput, 0)
		for i := range value {
			set[value[i].TypeID] = true
			if value[i].IsInput == 0 {
				output = recipeInput{value[i].TypeID, value[i].Quantity}
			} else {
				input = append(input, recipeInput{value[i].TypeID, value[i].Quantity})
			}
		}
		insertableRecipes = append(insertableRecipes, piRecipe{output.TypeID, output.Quantity, input})
		// fmt.Println("LINE: ", insertableRecipes[len(insertableRecipes)-1])
	}
	// fmt.Println(insertableRecipes)

	pidb := mdb.DB("sde").C("planetschematicrecipes")
	pidb.RemoveAll(nil)
	for x := range insertableRecipes {
		// fmt.Println(insertableRecipes[x])
		m_err := pidb.Insert(&insertableRecipes[x])
		checkErr(m_err)
	}
	return set
}

/*
Scrapify and its stuff
*/
type itemRecipe struct {
	TypeId int
	Recipe []component
}

type component struct {
	MaterialTypeId int
	Quantity       int
}

func scrapify(sql *sql.DB, mdb *mgo.Session) map[int]bool {
	// fmt.Println("Inside 'scrapify'")

	set := make(map[int]bool)

	recipeMap := make(map[int][]component)
	recipeLines, err := sql.Query("select typeId, materialTypeId, quantity from invTypeMaterials")
	// recipeLines, err := sql.Query("select typeId, materialTypeId, quantity from invTypeMaterials where typeId in (626, 23527, 35658)")
	defer recipeLines.Close()
	if err != nil {
		log.Fatal(err)
	}
	/*
		it's easier to get a typeId's []component with a stopover at Map
	*/
	for recipeLines.Next() {
		var typeId int
		var materialTypeId int
		var quantity int
		recipeLines.Scan(&typeId, &materialTypeId, &quantity)

		if recipe, prs := recipeMap[typeId]; prs {
			recipe = append(recipe, component{materialTypeId, quantity})
			recipeMap[typeId] = recipe
		} else {
			recipe := make([]component, 0)
			recipe = append(recipe, component{materialTypeId, quantity})
			recipeMap[typeId] = recipe
		}
	}
	// fmt.Println("RECIPEMAP GOES HERE", recipeMap, "\n\n\n")

	/*
		turn Map into itemRecipe
	*/
	insertableRecipes := make([]itemRecipe, 0)
	var key int
	var value []component
	for key, value = range recipeMap {
		insertableRecipes = append(insertableRecipes, itemRecipe{key, value})
		set[key] = true
		for i := range value {
			set[value[i].MaterialTypeId] = true
		}
	}

	recipedb := mdb.DB("sde").C("scraprecipes")
	recipedb.RemoveAll(nil)
	//can be done with an []interface{}, whatever that means
	// var i int = 0
	for x := range insertableRecipes {
		// fmt.Println(i)
		// i += 1
		m_err := recipedb.Insert(&insertableRecipes[x])
		checkErr(m_err)
	}
	// fmt.Println("Inserted", i, "item recipes!")
	// fmt.Println("Found unique typeIDs:", len(set))
	return set
}

func checkErr(err error) {
	if err != nil {
		// log.Fatal(err)
		fmt.Println(err)
		panic(err)
	}
}
