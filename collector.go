package main

import (
	"context"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/prometheus/client_golang/prometheus"
)

type ECRCollector struct {
	client *ecr.Client

	// Metrics
	repoCount          *prometheus.Desc
	imageCount         *prometheus.Desc
	imageSizeMax       *prometheus.Desc
	imageSizeMin       *prometheus.Desc
	imageSizeAvg       *prometheus.Desc
	latestPushTime     *prometheus.Desc
	latestPullTime     *prometheus.Desc
	scrapeErrors       *prometheus.Desc
	scrapeDuration     *prometheus.Desc
}

func NewECRCollector(client *ecr.Client) *ECRCollector {
	return &ECRCollector{
		client: client,
		repoCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "repositories_total"),
			"Total number of ECR repositories",
			nil,
			nil,
		),
		imageCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "images_total"),
			"Number of images in ECR repository",
			[]string{"repository_name", "repository_uri"},
			nil,
		),
		imageSizeMax: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "image_size_max_bytes"),
			"Maximum image size in repository (bytes)",
			[]string{"repository_name", "repository_uri"},
			nil,
		),
		imageSizeMin: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "image_size_min_bytes"),
			"Minimum image size in repository (bytes)",
			[]string{"repository_name", "repository_uri"},
			nil,
		),
		imageSizeAvg: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "image_size_avg_bytes"),
			"Average image size in repository (bytes)",
			[]string{"repository_name", "repository_uri"},
			nil,
		),
		latestPushTime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "latest_push_timestamp"),
			"Timestamp of latest image push",
			[]string{"repository_name", "repository_uri"},
			nil,
		),
		latestPullTime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "latest_pull_timestamp"),
			"Timestamp of latest image pull",
			[]string{"repository_name", "repository_uri"},
			nil,
		),
		scrapeErrors: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "scrape_errors_total"),
			"Total number of scrape errors",
			nil,
			nil,
		),
		scrapeDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "scrape_duration_seconds"),
			"Duration of the scrape",
			nil,
			nil,
		),
	}
}

func (c *ECRCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.repoCount
	ch <- c.imageCount
	ch <- c.imageSizeMax
	ch <- c.imageSizeMin
	ch <- c.imageSizeAvg
	ch <- c.latestPushTime
	ch <- c.latestPullTime
	ch <- c.scrapeErrors
	ch <- c.scrapeDuration
}

func (c *ECRCollector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	errorCount := 0

	log.Info("Starting metrics collection")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get all repositories
	log.Info("Fetching ECR repositories...")
	repos, err := c.getAllRepositories(ctx)
	if err != nil {
		log.Errorf("Failed to get repositories: %v", err)
		errorCount++
		// Still send error metrics even if we can't get repos
		ch <- prometheus.MustNewConstMetric(
			c.repoCount,
			prometheus.GaugeValue,
			0,
		)
	} else {
		log.Infof("Found %d repositories", len(repos))
		// Send total repository count
		ch <- prometheus.MustNewConstMetric(
			c.repoCount,
			prometheus.GaugeValue,
			float64(len(repos)),
		)

		// Process each repository
		for i, repo := range repos {
			log.Infof("Processing repository %d/%d: %s", i+1, len(repos), *repo.RepositoryName)
			c.collectRepositoryMetrics(ctx, repo, ch, &errorCount)
			log.Infof("Completed repository %d/%d: %s", i+1, len(repos), *repo.RepositoryName)
		}
	}

	log.Info("Sending final scrape metrics...")
	// Send scrape metrics
	ch <- prometheus.MustNewConstMetric(
		c.scrapeErrors,
		prometheus.CounterValue,
		float64(errorCount),
	)

	ch <- prometheus.MustNewConstMetric(
		c.scrapeDuration,
		prometheus.GaugeValue,
		time.Since(start).Seconds(),
	)

	log.Infof("Metrics collection completed in %.2f seconds with %d errors", time.Since(start).Seconds(), errorCount)
}

func (c *ECRCollector) getAllRepositories(ctx context.Context) ([]types.Repository, error) {
	var allRepos []types.Repository
	var nextToken *string

	log.Debug("Starting to fetch repositories")
	
	for {
		input := &ecr.DescribeRepositoriesInput{
			NextToken: nextToken,
		}

		log.Debug("Making DescribeRepositories API call")
		result, err := c.client.DescribeRepositories(ctx, input)
		if err != nil {
			log.Errorf("DescribeRepositories API call failed: %v", err)
			return nil, err
		}

		log.Debugf("Got %d repositories in this batch", len(result.Repositories))
		allRepos = append(allRepos, result.Repositories...)

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	log.Debugf("Total repositories fetched: %d", len(allRepos))
	return allRepos, nil
}

func (c *ECRCollector) collectRepositoryMetrics(ctx context.Context, repo types.Repository, ch chan<- prometheus.Metric, errorCount *int) {
	// Add nil checks
	if repo.RepositoryName == nil {
		log.Error("Repository name is nil, skipping")
		*errorCount++
		return
	}
	if repo.RepositoryUri == nil {
		log.Errorf("Repository URI is nil for repo %s, skipping", *repo.RepositoryName)
		*errorCount++
		return
	}

	repoName := *repo.RepositoryName
	repoURI := *repo.RepositoryUri

	log.Debugf("Starting collectRepositoryMetrics for: %s", repoName)

	labels := []string{repoName, repoURI}

	log.Debugf("Fetching images for repository: %s", repoName)
	// Get images for this repository
	images, err := c.getRepositoryImages(ctx, repoName)
	if err != nil {
		log.Errorf("Failed to get images for repository %s: %v", repoName, err)
		*errorCount++
		// Still send zero count for this repo
		log.Debugf("Sending zero image count metric for repository: %s", repoName)
		ch <- prometheus.MustNewConstMetric(
			c.imageCount,
			prometheus.GaugeValue,
			0,
			labels...,
		)
		log.Debugf("Finished processing repository %s (with error)", repoName)
		return
	}

	log.Debugf("Found %d images in repository %s", len(images), repoName)

	// Image count
	log.Debugf("Sending image count metric for repository: %s", repoName)
	ch <- prometheus.MustNewConstMetric(
		c.imageCount,
		prometheus.GaugeValue,
		float64(len(images)),
		labels...,
	)
	log.Debugf("Image count metric sent for repository: %s", repoName)

	if len(images) == 0 {
		log.Debugf("No images found, finishing repository: %s", repoName)
		return
	}

	// Calculate size metrics
	var sizes []int64
	var latestPush, latestPull time.Time

	for _, image := range images {
		if image.ImageSizeInBytes != nil {
			sizes = append(sizes, *image.ImageSizeInBytes)
		}

		if image.ImagePushedAt != nil && image.ImagePushedAt.After(latestPush) {
			latestPush = *image.ImagePushedAt
		}

		if image.LastRecordedPullTime != nil && image.LastRecordedPullTime.After(latestPull) {
			latestPull = *image.LastRecordedPullTime
		}
	}

	// Size statistics
	if len(sizes) > 0 {
		sort.Slice(sizes, func(i, j int) bool { return sizes[i] < sizes[j] })

		minSize := float64(sizes[0])
		maxSize := float64(sizes[len(sizes)-1])

		var totalSize int64
		for _, size := range sizes {
			totalSize += size
		}
		avgSize := float64(totalSize) / float64(len(sizes))

		ch <- prometheus.MustNewConstMetric(
			c.imageSizeMin,
			prometheus.GaugeValue,
			minSize,
			labels...,
		)

		ch <- prometheus.MustNewConstMetric(
			c.imageSizeMax,
			prometheus.GaugeValue,
			maxSize,
			labels...,
		)

		ch <- prometheus.MustNewConstMetric(
			c.imageSizeAvg,
			prometheus.GaugeValue,
			avgSize,
			labels...,
		)
	}

	// Latest push time
	if !latestPush.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.latestPushTime,
			prometheus.GaugeValue,
			float64(latestPush.Unix()),
			labels...,
		)
	}

	// Latest pull time
	if !latestPull.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.latestPullTime,
			prometheus.GaugeValue,
			float64(latestPull.Unix()),
			labels...,
		)
	}

	log.Debugf("Finished processing repository: %s", repoName)
}

func (c *ECRCollector) getRepositoryImages(ctx context.Context, repoName string) ([]types.ImageDetail, error) {
	var allImages []types.ImageDetail
	var nextToken *string

	log.Debugf("Starting to fetch images for repository: %s", repoName)
	
	for {
		input := &ecr.DescribeImagesInput{
			RepositoryName: &repoName,
			NextToken:      nextToken,
		}

		log.Debugf("Making DescribeImages API call for repository: %s", repoName)
		result, err := c.client.DescribeImages(ctx, input)
		if err != nil {
			log.Errorf("DescribeImages API call failed for repository %s: %v", repoName, err)
			return nil, err
		}

		log.Debugf("Got %d images in this batch for repository: %s", len(result.ImageDetails), repoName)
		allImages = append(allImages, result.ImageDetails...)

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	log.Debugf("Total images fetched for repository %s: %d", repoName, len(allImages))
	return allImages, nil
}