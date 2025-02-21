package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/playwrightLocatr"
	"github.com/vertexcover-io/locatr/golang/reranker"
)

func GenerateInputSchema[T any]() (map[string]interface{}, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var v T
	schema := reflector.Reflect(v)

	// Marshal the schema to JSON
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Unmarshal into anthropic.BetaToolInputSchemaParam
	var result interface{}
	if err := json.Unmarshal(schemaBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal into BetaToolInputSchemaParam: %w", err)
	}

	return result.(map[string]interface{}), nil
}

type Point struct {
	X float64 `jsonschema_description:"The X-coordinate value"`
	Y float64 `jsonschema_description:"The Y-coordinate value"`
}

type Output struct {
	GeneratedPoints *[]Point
	InputTokens     int64
	OutputTokens    int64
	TotalTokens     int64
}

type Captured struct {
	Url                string
	ScrollCoordinates  Point
	ElementDescription string
	ElementCoordinates Point
}

type LocatrInterface interface {
	call(page *playwright.Page, query string) (*Output, error)
}

type OriginalLocatr struct {
	instance *playwrightLocatr.PlaywrightLocator
}

func (l OriginalLocatr) call(page *playwright.Page, query string) (*Output, error) {
	_, err := l.instance.GetLocatr(query)
	if err != nil {
		return nil, err
	}

	results := l.instance.GetLocatrResults()
	lastResult := results[len(results)-1]
	output := Output{
		GeneratedPoints: nil,
		InputTokens:     int64(lastResult.InputTokens),
		OutputTokens:    int64(lastResult.OutputTokens),
		TotalTokens:     int64(lastResult.TotalTokens),
	}

	if len(lastResult.AllLocatrs) == 0 {
		return &output, nil
	}

	points := make([]Point, len(lastResult.AllLocatrs))
	appended := false
	viewportSize := (*page).ViewportSize()

	for _, value := range lastResult.AllLocatrs {
		bbox, err := (*page).Locator(value).BoundingBox()
		if err != nil {
			continue
		}
		X := bbox.X + bbox.Width/2
		Y := bbox.Y + bbox.Height/2

		// Check if the element is within the viewport
		if X < 0 || Y < 0 ||
			bbox.X > float64(viewportSize.Width) || bbox.Y > float64(viewportSize.Height) {
			continue // Skip elements outside the viewport
		}

		points = append(points, Point{X: X, Y: Y})
		appended = true
	}

	if appended {
		output.GeneratedPoints = &points
		return &output, nil
	}
	return nil, errors.New("no valid locatrs found")
}

const ANTHROPIC_GROUNDING_INSTRUCTION string = `Given the screen resolution of 1280x800, identify the exact (X, Y) coordinates for the described area, element, or object on a browser GUI screen.`

type AnthropicGroundingLocatr struct {
	client *anthropic.Client
}

func (l AnthropicGroundingLocatr) call(page *playwright.Page, query string) (*Output, error) {

	imageBytes, err := (*page).Screenshot(
		playwright.PageScreenshotOptions{Type: playwright.ScreenshotTypeJpeg},
	)
	if err != nil {
		return nil, err
	}

	toolInputSchema, err := GenerateInputSchema[Point]()
	if err != nil {
		return nil, err
	}
	fmt.Printf("Tool input schema: %v\n", toolInputSchema)
	anthropicToolInputSchema := anthropic.BetaToolInputSchemaParam{
		Type:       anthropic.F(anthropic.BetaToolInputSchemaTypeObject),
		Properties: anthropic.F(toolInputSchema["properties"]),
	}

	response, err := l.client.Beta.Messages.New(
		context.TODO(),
		anthropic.BetaMessageNewParams{
			Model:     anthropic.F(anthropic.ModelClaude3_5SonnetLatest),
			MaxTokens: anthropic.F(int64(1024)),
			Messages: anthropic.F([]anthropic.BetaMessageParam{
				{
					Role: anthropic.F(anthropic.BetaMessageParamRoleUser),
					Content: anthropic.F([]anthropic.BetaContentBlockParamUnion{
						anthropic.BetaTextBlockParam{
							Type: anthropic.F(anthropic.BetaTextBlockParamTypeText),
							Text: anthropic.String(ANTHROPIC_GROUNDING_INSTRUCTION),
						},
						anthropic.BetaTextBlockParam{
							Type: anthropic.F(anthropic.BetaTextBlockParamTypeText),
							Text: anthropic.String(fmt.Sprintf("Description: %s", query)),
						},
						anthropic.BetaImageBlockParam{
							Type: anthropic.F(anthropic.BetaImageBlockParamTypeImage),
							Source: anthropic.F(
								anthropic.BetaImageBlockParamSource{
									Type:      anthropic.F(anthropic.BetaImageBlockParamSourceTypeBase64),
									MediaType: anthropic.F(anthropic.BetaImageBlockParamSourceMediaTypeImageJPEG),
									Data:      anthropic.F(base64.StdEncoding.EncodeToString(imageBytes)),
								},
							),
						},
					}),
				},
			}),
			Tools: anthropic.F([]anthropic.BetaToolUnionUnionParam{
				anthropic.BetaToolParam{
					Name:        anthropic.String("return_coordinates"),
					Description: anthropic.String("Return the coordinates"),
					InputSchema: anthropic.F(anthropicToolInputSchema),
				},
			}),
			ToolChoice: anthropic.F(anthropic.BetaToolChoiceUnionParam(
				anthropic.BetaToolChoiceToolParam{
					Type:                   anthropic.F(anthropic.BetaToolChoiceToolTypeTool),
					Name:                   anthropic.String("return_coordinates"),
					DisableParallelToolUse: anthropic.Bool(true),
				},
			)),
			Betas: anthropic.F([]string{anthropic.AnthropicBetaComputerUse2024_10_22}),
		},
	)
	if err != nil {
		fmt.Printf("Error creating message: %v\n", err)
		return nil, err
	}

	output := Output{
		GeneratedPoints: nil,
		InputTokens:     response.Usage.InputTokens,
		OutputTokens:    response.Usage.OutputTokens,
		TotalTokens:     response.Usage.InputTokens + response.Usage.OutputTokens,
	}
	content := response.Content[0]

	if content.Type == anthropic.BetaContentBlockTypeToolUse {
		fmt.Printf("Arguments: %v\n", content.Input)
		input := content.Input.(map[string]interface{})

		convertToFloat := func(value interface{}) (float64, error) {
			switch v := value.(type) {
			case float64:
				return v, nil
			case string:
				// Regex to match numeric part (supports integers, decimals, and negative numbers)
				re := regexp.MustCompile(`^-?\d*\.?\d+`)
				match := re.FindString(v)
				if match == "" {
					return 0, fmt.Errorf("no numeric part found in string: %s", v)
				}
				return strconv.ParseFloat(match, 64)
			default:
				return 0, fmt.Errorf("unexpected type %T", v)
			}
		}

		x, err := convertToFloat(input["X"])
		if err != nil {
			return nil, fmt.Errorf("invalid X value: %v", err)
		}

		y, err := convertToFloat(input["Y"])
		if err != nil {
			return nil, fmt.Errorf("invalid Y value: %v", err)
		}

		point := Point{X: x, Y: y}
		output.GeneratedPoints = &[]Point{point}
		return &output, nil
	}

	fmt.Printf("Raw response: %s", content.Text)
	return &output, fmt.Errorf("no tool call found, raw text response: %s", content.Text)

}

func getOriginalLocatrInstance(page *playwright.Page, rerank bool) *playwrightLocatr.PlaywrightLocator {
	llmClient, err := llm.NewLlmClient(
		llm.OpenAI, "gpt-4o", os.Getenv("OPENAI_API_KEY"),
	)
	if err != nil {
		logger.Logger.Info("Error creating LLM client", "error", err)
		return nil
	}
	locatrOptions := locatr.BaseLocatrOptions{LlmClient: llmClient}
	if rerank {
		reRankClient := reranker.NewCohereClient(os.Getenv("COHERE_API_KEY"))
		locatrOptions.ReRankClient = reRankClient
	}
	return playwrightLocatr.NewPlaywrightLocatr(*page, locatrOptions)
}

// Convert []byte to image.Image
func bytesToImage(imgBytes []byte) (draw.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, err
	}

	// Convert to RGBA for drawing
	rgbaImg := image.NewRGBA(img.Bounds())
	draw.Draw(rgbaImg, img.Bounds(), img, image.Point{}, draw.Src)
	return rgbaImg, nil
}

// Draw points with color & radius on image
func drawPoints(img draw.Image, points *[]Point, clr color.Color, radius int) {
	for _, pt := range *points {
		for dx := -radius; dx <= radius; dx++ {
			for dy := -radius; dy <= radius; dy++ {
				if dx*dx+dy*dy <= radius*radius { // Circle formula
					X := int(pt.X) + dx
					Y := int(pt.Y) + dy
					if image.Pt(X, Y).In(img.Bounds()) {
						img.Set(X, Y, clr)
					}
				}
			}
		}
	}
}

// Convert image.Image back to []byte
func imageToBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, nil)
	return buf.Bytes(), err
}

func processURLs(urls []string) error {

	if err := os.MkdirAll("images", 0755); err != nil {
		log.Fatal("Failed to create directory:", err)
	}

	err := playwright.Install()
	if err != nil {
		log.Fatal(err.Error())
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start Playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %v", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  1280,
			Height: 800,
		},
		BypassCSP: playwright.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("could not create browser context: %v", err)
	}

	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("could not create new page: %v", err)
	}

	// Handle CTRL+C for graceful exit
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	timestamp := time.Now().Format("20060102_150405")
	outputPath := fmt.Sprintf("results_%s.jsonl", timestamp)

	for _, url := range urls {
		fmt.Printf("Navigating to: %s\n", url)
		if _, err := page.Goto(url); err != nil {
			return fmt.Errorf("failed to load URL '%s': %v", url, err)
		}

		if _, err := page.AddScriptTag(playwright.PageAddScriptTagOptions{
			Path: playwright.String("inject.js"),
		}); err != nil {
			return fmt.Errorf("failed to inject script: %v", err)
		}

		if _, err := page.Evaluate(`() => window.elementSelector.createInputBox()`); err != nil {
			return fmt.Errorf("failed to create input box: %v", err)
		}

		if err := page.WaitForLoadState(
			playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded},
		); err != nil {
			return fmt.Errorf("failed waiting for load state: %v", err)
		}

		fmt.Println("Press ENTER to process all captured elements (Ctrl+C to exit)...")
		go func() {
			bufio.NewReader(os.Stdin).ReadBytes('\n')
			sigs <- syscall.SIGINT
		}()

		<-sigs // Wait for user input or Ctrl+C

		if _, err := page.Evaluate(`() => window.elementSelector.destroyInputBox()`); err != nil {
			return fmt.Errorf("failed to destroy input box: %v", err)
		}

		jsHandle, err := page.WaitForFunction(`() => window.Captured`, []any{})
		if err != nil {
			return fmt.Errorf("failed to get captured elements: %v", err)
		}

		interfaceValue, err := jsHandle.JSONValue()
		if err != nil {
			return fmt.Errorf("failed to get captured elements: %v", err)
		}

		// Convert capturedElements to JSON and then to []Captured
		jsonBytes, err := json.Marshal(interfaceValue)
		if err != nil {
			return fmt.Errorf("failed to marshal captured elements: %v", err)
		}

		var captured []Captured
		if err := json.Unmarshal(jsonBytes, &captured); err != nil {
			return fmt.Errorf("failed to unmarshal captured elements: %v", err)
		}

		originalLocatrWithoutReranking := OriginalLocatr{instance: getOriginalLocatrInstance(&page, false)} // without Reranker
		originalLocatrWithReranking := OriginalLocatr{instance: getOriginalLocatrInstance(&page, true)}
		anthropicGroundingLocatr := AnthropicGroundingLocatr{
			client: anthropic.NewClient(option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY"))),
		}

		redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}      // Selected manually
		blueColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}     // Generated by original Locator (without reranking)
		yellowColor := color.RGBA{R: 255, G: 255, B: 0, A: 255} // Generated by original Locator (with reranking)
		greenColor := color.RGBA{R: 0, G: 255, B: 0, A: 255}    // Generated by anthropic grounding locatr

		// Process Captured Elements
		for _, elem := range captured {
			scrollCoords := elem.ScrollCoordinates
			fmt.Printf("Scrolling to: X=%.2f, Y=%.2f\n", scrollCoords.X, scrollCoords.Y)
			if _, err := page.Evaluate(
				`([X, Y]) => window.scrollTo(X, Y)`, []float64{scrollCoords.X, scrollCoords.Y},
			); err != nil {
				return fmt.Errorf("failed to scroll: %v", err)
			}

			// Wait until scroll reaches the desired point
			if _, err := page.WaitForFunction(
				`([X, Y]) => window.scrollX == X && window.scrollY == Y`, []float64{scrollCoords.X, scrollCoords.Y},
			); err != nil {
				return fmt.Errorf("scroll verification failed: %v", err)
			}

			fmt.Printf("Scrolled to: X=%.2f, Y=%.2f\n", scrollCoords.X, scrollCoords.Y)
			fmt.Printf("Element Description: %s\n", elem.ElementDescription)

			ssBytes, err := page.Screenshot(
				playwright.PageScreenshotOptions{Type: playwright.ScreenshotTypeJpeg},
			)
			if err != nil {
				return err
			}

			// Convert bytes to image
			img, err := bytesToImage(ssBytes)
			if err != nil {
				log.Fatalf("Failed to decode image: %v", err)
			}

			drawPoints(img, &[]Point{elem.ElementCoordinates}, redColor, 14)
			outputs := make(map[string]*Output, 3)

			call := func(locatr LocatrInterface, name string, color color.Color, radius int) {
				fmt.Printf("\nCalling %s\n", name)
				output, err := locatr.call(&page, elem.ElementDescription)
				if err != nil {
					fmt.Printf("Error occured calling `%s`: %v\n", name, err)
				} else {
					fmt.Printf("Output of %s: %v\n", name, *output)
					generatedPoints := (*output).GeneratedPoints
					if generatedPoints == nil {
						fmt.Printf("Could not extract valid locator using `%s`", name)
					} else {
						drawPoints(img, generatedPoints, color, radius)
					}
					outputs[name] = output
				}
			}

			call(originalLocatrWithoutReranking, "originalLocatrWithoutReranking", blueColor, 12)
			call(originalLocatrWithReranking, "originalLocatrWithReranking", yellowColor, 10)
			call(anthropicGroundingLocatr, "anthropicGroundingLocatr", greenColor, 8)

			// Convert back to bytes
			finalBytes, err := imageToBytes(img)
			if err != nil {
				log.Fatalf("Failed to encode image: %v", err)
			}

			// Save final image
			imagePath := fmt.Sprintf("images/%s.jpeg", uuid.NewString())
			err = os.WriteFile(imagePath, finalBytes, 0644)
			if err != nil {
				log.Fatalf("Failed to save image: %v", err)
			}

			log.Printf("Image saved as %s", imagePath)

			data := struct {
				Captured  // Embedded to flatten fields
				Outputs   map[string]*Output
				ImagePath string
			}{elem, outputs, imagePath}

			// Marshal entry to JSON
			dataBytes, err := json.Marshal(data)
			if err != nil {
				log.Fatalf("Failed to marshal entry: %v", err)
			}

			// Open file in append mode
			file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("Failed to open JSONL file: %v", err)
			}
			defer file.Close()

			// Write the JSON line
			if _, err := file.Write(append(dataBytes, '\n')); err != nil {
				log.Fatalf("Failed to write to JSONL file: %v", err)
			}
		}
	}
	return nil
}

// loadURLs loads URLs from the given input file, one per line.
func loadURLs(inputPath string) ([]string, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file")
	}

	// Command-line flag for input file
	inputFile := flag.String("input", "", "Path to input file containing URLs")
	flag.Parse()

	if *inputFile == "" {
		log.Fatal("The -input flag is required")
	}

	// Load URLs from the input file
	urls, err := loadURLs(*inputFile)
	if err != nil {
		log.Fatalf("Error loading URLs: %v", err)
	}

	// Process URLs and write to output file
	if err := processURLs(urls); err != nil {
		log.Fatalf("Error processing URLs: %v", err)
	}

	log.Println("Processing completed successfully")
}
