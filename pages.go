package main

import "fmt"

func RenderPreparingPage(subdomain string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta http-equiv="refresh" content="3">
    <title>Preparing %s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: #f5f5f5;
            color: #333;
        }
        .container {
            text-align: center;
            padding: 2rem;
        }
        .spinner {
            width: 40px;
            height: 40px;
            border: 4px solid #e0e0e0;
            border-top-color: #333;
            border-radius: 50%%;
            animation: spin 0.8s linear infinite;
            margin: 0 auto 1.5rem;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        h1 { font-size: 1.5rem; font-weight: 500; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="spinner"></div>
        <h1>Preparing environment</h1>
        <p>Setting up <strong>%s</strong>. This page will refresh automatically.</p>
    </div>
</body>
</html>`, subdomain, subdomain)
}

func RenderNotFoundPage(subdomain string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Not Found - %s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: #f5f5f5;
            color: #333;
        }
        .container {
            text-align: center;
            padding: 2rem;
        }
        h1 { font-size: 1.5rem; font-weight: 500; }
        p { color: #666; }
        code {
            background: #e8e8e8;
            padding: 0.2rem 0.5rem;
            border-radius: 3px;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Image does not exist</h1>
        <p>No Docker image was found for <code>%s</code>.</p>
        <p>Make sure your CI pipeline has built and pushed the image for this branch.</p>
    </div>
</body>
</html>`, subdomain, subdomain)
}
