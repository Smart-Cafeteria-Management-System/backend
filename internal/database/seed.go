package database

import (
	"log"
	"time"

	"github.com/smart-cafeteria/backend/internal/models"
	"gorm.io/gorm"
)

// Seed populates the database with initial data
func Seed(db *gorm.DB) error {
	log.Println("Checking database seeding status...")

	// 1. Seed Users
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount == 0 {
		log.Println("Seeding users...")
		// Create admin user
		admin := models.User{
			Name:                "Admin",
			Email:               "admin@cafeteria.com",
			Password:            "admin123",
			Role:                models.RoleAdmin,
			NotificationEnabled: true,
		}
		if err := db.Create(&admin).Error; err != nil {
			return err
		}

		// Create student user
		studentID := "STU001"
		student := models.User{
			Name:                "John Keller",
			Email:               "john.keller@university.edu",
			Password:            "john123",
			Role:                models.RoleStudent,
			StudentID:           &studentID,
			DietaryRestrictions: `["vegetarian"]`,
			NotificationEnabled: true,
		}
		if err := db.Create(&student).Error; err != nil {
			return err
		}

		// Create staff user
		staff := models.User{
			Name:                "Staff User",
			Email:               "staff@cafeteria.com",
			Password:            "staff123",
			Role:                models.RoleStaff,
			NotificationEnabled: true,
		}
		if err := db.Create(&staff).Error; err != nil {
			return err
		}
	}

	// 2. Seed Menu Items
	var menuCount int64
	db.Model(&models.MenuItem{}).Count(&menuCount)
	if menuCount == 0 {
		log.Println("Seeding menu items...")
		menuItems := []models.MenuItem{
			{
				Name:               "Masala Dosa",
				Category:           models.CategoryMain,
				Price:              60,
				Calories:           intPtr(350),
				Protein:            intPtr(8),
				Carbs:              intPtr(55),
				Fat:                intPtr(12),
				Available:          true,
				SustainabilityScore: 4,
				PreparationTime:    10,
			},
			{
				Name:               "Idli Sambar",
				Category:           models.CategoryMain,
				Price:              40,
				Calories:           intPtr(200),
				Protein:            intPtr(6),
				Carbs:              intPtr(40),
				Fat:                intPtr(2),
				Available:          true,
				SustainabilityScore: 5,
				PreparationTime:    5,
			},
			{
				Name:               "Paneer Butter Masala",
				Category:           models.CategoryMain,
				Price:              120,
				Calories:           intPtr(450),
				Protein:            intPtr(18),
				Carbs:              intPtr(25),
				Fat:                intPtr(32),
				Available:          true,
				SustainabilityScore: 3,
				PreparationTime:    15,
			},
			{
				Name:               "Dal Tadka",
				Category:           models.CategoryMain,
				Price:              80,
				Calories:           intPtr(250),
				Protein:            intPtr(12),
				Carbs:              intPtr(35),
				Fat:                intPtr(8),
				Available:          true,
				SustainabilityScore: 5,
				PreparationTime:    10,
			},
			{
				Name:               "Vegetable Biryani",
				Category:           models.CategoryMain,
				Price:              100,
				Calories:           intPtr(400),
				Protein:            intPtr(10),
				Carbs:              intPtr(65),
				Fat:                intPtr(12),
				Available:          true,
				SustainabilityScore: 4,
				PreparationTime:    20,
			},
			{
				Name:               "Roti",
				Category:           models.CategorySide,
				Price:              10,
				Calories:           intPtr(100),
				Protein:            intPtr(3),
				Carbs:              intPtr(20),
				Fat:                intPtr(1),
				Available:          true,
				SustainabilityScore: 5,
				PreparationTime:    3,
			},
			{
				Name:               "Steamed Rice",
				Category:           models.CategorySide,
				Price:              30,
				Calories:           intPtr(200),
				Protein:            intPtr(4),
				Carbs:              intPtr(45),
				Fat:                intPtr(0),
				Available:          true,
				SustainabilityScore: 4,
				PreparationTime:    5,
			},
			{
				Name:               "Masala Chai",
				Category:           models.CategoryBeverage,
				Price:              15,
				Calories:           intPtr(80),
				Protein:            intPtr(2),
				Carbs:              intPtr(12),
				Fat:                intPtr(3),
				Available:          true,
				SustainabilityScore: 4,
				PreparationTime:    3,
			},
			{
				Name:               "Fresh Lime Soda",
				Category:           models.CategoryBeverage,
				Price:              25,
				Calories:           intPtr(60),
				Protein:            intPtr(0),
				Carbs:              intPtr(15),
				Fat:                intPtr(0),
				Available:          true,
				SustainabilityScore: 5,
				PreparationTime:    2,
			},
			{
				Name:               "Gulab Jamun",
				Category:           models.CategoryDessert,
				Price:              35,
				Calories:           intPtr(150),
				Protein:            intPtr(2),
				Carbs:              intPtr(25),
				Fat:                intPtr(6),
				Available:          true,
				SustainabilityScore: 3,
				PreparationTime:    5,
			},
		}

		for _, item := range menuItems {
			db.Create(&item)
		}
	}

	// 3. Seed Meal Slots
	var slotCount int64
	db.Model(&models.MealSlot{}).Count(&slotCount)
	if slotCount == 0 {
		log.Println("Seeding meal slots...")
		today := time.Now().Truncate(24 * time.Hour)
		for i := 0; i < 7; i++ {
			date := today.AddDate(0, 0, i)
			slots := []models.MealSlot{
				{
					Date:      date,
					MealType:  models.MealBreakfast,
					StartTime: "07:00",
					EndTime:   "09:00",
					Capacity:  50,
					Status:    models.SlotAvailable,
				},
				{
					Date:      date,
					MealType:  models.MealLunch,
					StartTime: "12:00",
					EndTime:   "14:00",
					Capacity:  100,
					Status:    models.SlotAvailable,
				},
				{
					Date:      date,
					MealType:  models.MealDinner,
					StartTime: "19:00",
					EndTime:   "21:00",
					Capacity:  75,
					Status:    models.SlotAvailable,
				},
			}
			for _, slot := range slots {
				db.Create(&slot)
			}
		}
	}

	// 4. Seed Demand Forecasts
	var forecastCount int64
	db.Model(&models.DemandForecast{}).Count(&forecastCount)
	if forecastCount < 10 { // Force seed if sparse or empty
		log.Println("Seeding demand forecasts (table is empty or sparse)...")
		today := time.Now().Truncate(24 * time.Hour)
		weathers := []models.WeatherCondition{
			models.WeatherSunny, models.WeatherCloudy, models.WeatherRainy, models.WeatherSunny,
			models.WeatherSunny, models.WeatherCloudy, models.WeatherSunny,
		}
		schedules := []models.AcademicSchedule{
			models.ScheduleRegular, models.ScheduleRegular, models.ScheduleExams, models.ScheduleRegular,
			models.ScheduleRegular, models.ScheduleRegular, models.ScheduleRegular,
		}

		// Past 14 days
		for i := 14; i >= 1; i-- {
			date := today.AddDate(0, 0, -i)
			dayIdx := i % 7
			forecasts := []models.DemandForecast{
				{
					Date:             date,
					MealType:         models.MealBreakfast,
					PredictedDemand:  70 + (i % 20),
					ActualDemand:     65 + (i % 25),
					Confidence:       80 + (i % 15),
					WeatherCondition: weathers[dayIdx],
					AcademicSchedule: schedules[dayIdx],
				},
				{
					Date:             date,
					MealType:         models.MealLunch,
					PredictedDemand:  140 + (i % 30),
					ActualDemand:     135 + (i % 35),
					Confidence:       85 + (i % 10),
					WeatherCondition: weathers[dayIdx],
					AcademicSchedule: schedules[dayIdx],
				},
				{
					Date:             date,
					MealType:         models.MealDinner,
					PredictedDemand:  110 + (i % 25),
					ActualDemand:     105 + (i % 30),
					Confidence:       82 + (i % 12),
					WeatherCondition: weathers[dayIdx],
					AcademicSchedule: schedules[dayIdx],
				},
			}
			for _, f := range forecasts {
				db.Create(&f)
			}
		}

		// Future 7 days
		daysMap := map[time.Weekday]models.DayOfWeek{
			time.Monday:    models.Monday,
			time.Tuesday:   models.Tuesday,
			time.Wednesday: models.Wednesday,
			time.Thursday:  models.Thursday,
			time.Friday:    models.Friday,
			time.Saturday:  models.Saturday,
			time.Sunday:    models.Sunday,
		}
		for i := 0; i < 7; i++ {
			date := today.AddDate(0, 0, i)
			dayOfWeek := daysMap[date.Weekday()]
			baseDemand := map[models.MealType]int{
				models.MealBreakfast: 80,
				models.MealLunch:     150,
				models.MealDinner:    120,
			}
			if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
				for k := range baseDemand {
					baseDemand[k] = baseDemand[k] / 2
				}
			}
			forecasts := []models.DemandForecast{
				{
					Date:             date,
					MealType:         models.MealBreakfast,
					PredictedDemand:  baseDemand[models.MealBreakfast] + (i * 3),
					Confidence:       85 - (i * 2),
					WeatherCondition: weathers[i],
					AcademicSchedule: schedules[i],
					DayOfWeek:        dayOfWeek,
				},
				{
					Date:             date,
					MealType:         models.MealLunch,
					PredictedDemand:  baseDemand[models.MealLunch] + (i * 5),
					Confidence:       88 - (i * 2),
					WeatherCondition: weathers[i],
					AcademicSchedule: schedules[i],
					DayOfWeek:        dayOfWeek,
				},
				{
					Date:             date,
					MealType:         models.MealDinner,
					PredictedDemand:  baseDemand[models.MealDinner] + (i * 4),
					Confidence:       83 - (i * 2),
					WeatherCondition: weathers[i],
					AcademicSchedule: schedules[i],
					DayOfWeek:        dayOfWeek,
				},
			}
			for _, f := range forecasts {
				db.Create(&f)
			}
		}
	}

	// 5. Seed Waste Logs
	var wasteCount int64
	db.Model(&models.WasteLog{}).Count(&wasteCount)
	if wasteCount == 0 {
		log.Println("Seeding waste logs...")
		today := time.Now().Truncate(24 * time.Hour)
		foodItems := []string{"Rice", "Dal", "Roti", "Paneer Curry", "Vegetable Biryani", "Idli", "Dosa"}
		var firstStaff models.User
		db.Where("role = ?", models.RoleStaff).First(&firstStaff)
		
		for i := 14; i >= 1; i-- {
			date := today.AddDate(0, 0, -i)
			wasteLogs := []models.WasteLog{
				{
					Date:             date,
					MealType:         models.MealLunch,
					Category:         models.WasteLeftover,
					FoodItem:         foodItems[i%len(foodItems)],
					PreparedQuantity: 150 + (i * 5),
					WastedQuantity:   15 + (i % 10),
					WasteWeight:      float64(2 + (i % 3)),
					Reason:           "Lower attendance than expected",
					RecordedBy:       firstStaff.ID,
				},
				{
					Date:             date,
					MealType:         models.MealDinner,
					Category:         models.WastePrepared,
					FoodItem:         foodItems[(i+3)%len(foodItems)],
					PreparedQuantity: 120 + (i * 4),
					WastedQuantity:   10 + (i % 8),
					WasteWeight:      float64(1.5 + float64(i%2)),
					Reason:           "Over-preparation",
					RecordedBy:       firstStaff.ID,
				},
			}
			for _, w := range wasteLogs {
				db.Create(&w)
			}
		}
	}

	// 6. Seed Free Add-ons
	var addonCount int64
	db.Model(&models.Addon{}).Count(&addonCount)
	if addonCount == 0 {
		log.Println("Seeding free add-ons...")
		addons := []models.Addon{
			// Beverages
			{
				Name:        "Masala Chai",
				Description: "A warm cup of spiced Indian tea with milk",
				PointsCost:  3,
				Category:    "beverage",
				Available:   true,
			},
			{
				Name:        "Filter Coffee",
				Description: "Traditional South Indian filter coffee",
				PointsCost:  4,
				Category:    "beverage",
				Available:   true,
			},
			{
				Name:        "Buttermilk",
				Description: "Refreshing spiced buttermilk (chaas)",
				PointsCost:  2,
				Category:    "beverage",
				Available:   true,
			},
			{
				Name:        "Fresh Lime Juice",
				Description: "Chilled lime juice with a hint of mint",
				PointsCost:  3,
				Category:    "beverage",
				Available:   true,
			},
			{
				Name:        "Mango Lassi",
				Description: "Creamy yogurt-based mango smoothie",
				PointsCost:  5,
				Category:    "beverage",
				Available:   true,
			},
			// Snacks
			{
				Name:        "Papad",
				Description: "Crispy roasted papad with pickle",
				PointsCost:  2,
				Category:    "snack",
				Available:   true,
			},
			{
				Name:        "Samosa",
				Description: "Crispy pastry filled with spiced potatoes",
				PointsCost:  5,
				Category:    "snack",
				Available:   true,
			},
			{
				Name:        "Vada",
				Description: "Crispy fried lentil donut with chutney",
				PointsCost:  4,
				Category:    "snack",
				Available:   true,
			},
			{
				Name:        "Bread Pakora",
				Description: "Deep-fried spiced bread fritters",
				PointsCost:  4,
				Category:    "snack",
				Available:   true,
			},
			{
				Name:        "Bhel Puri",
				Description: "Tangy puffed rice snack with vegetables",
				PointsCost:  3,
				Category:    "snack",
				Available:   true,
			},
			// Desserts
			{
				Name:        "Gulab Jamun",
				Description: "Soft milk dumplings soaked in sugar syrup",
				PointsCost:  5,
				Category:    "dessert",
				Available:   true,
			},
			{
				Name:        "Rasgulla",
				Description: "Soft cottage cheese balls in light syrup",
				PointsCost:  5,
				Category:    "dessert",
				Available:   true,
			},
			{
				Name:        "Jalebi",
				Description: "Crispy spiral-shaped sweet soaked in saffron syrup",
				PointsCost:  4,
				Category:    "dessert",
				Available:   true,
			},
			// Fruits
			{
				Name:        "Fresh Fruit Bowl",
				Description: "Seasonal fresh fruit mix (banana, apple, orange)",
				PointsCost:  3,
				Category:    "fruit",
				Available:   true,
			},
			{
				Name:        "Sliced Watermelon",
				Description: "Chilled watermelon slices - perfect for summer",
				PointsCost:  2,
				Category:    "fruit",
				Available:   true,
			},
		}

		for _, addon := range addons {
			db.Create(&addon)
		}
	}

	// 7. Seed Operating Hours
	var hoursCount int64
	db.Model(&models.OperatingHours{}).Count(&hoursCount)
	if hoursCount == 0 {
		log.Println("Seeding operating hours...")
		days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
		mealTypes := []models.MealType{models.MealBreakfast, models.MealLunch, models.MealDinner}

		for _, day := range days {
			for _, meal := range mealTypes {
				startTime := "08:00"
				endTime := "10:00"
				if meal == models.MealLunch {
					startTime = "12:00"
					endTime = "14:30"
				} else if meal == models.MealDinner {
					startTime = "19:00"
					endTime = "21:30"
				}

				db.Create(&models.OperatingHours{
					DayOfWeek: day,
					MealType:  meal,
					StartTime: startTime,
					EndTime:   endTime,
					IsClosed:  false,
				})
			}
		}
	}

	log.Println("Database seeding check completed successfully!")
	return nil
}

func intPtr(i int) *int {
	return &i
}
