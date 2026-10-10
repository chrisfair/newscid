package org.newscid;

import android.app.Activity;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;
import android.view.View;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.webkit.ValueCallback;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.FrameLayout;
import android.widget.ProgressBar;
import android.widget.TextView;
import android.widget.Toast;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.ServerSocket;
import java.net.URL;

public class MainActivity extends Activity {
    private static final String TAG = "NewScidApp";
    private static final int FILE_CHOOSER_REQUEST_CODE = 1001;

    private WebView mWebView;
    private FrameLayout mLoadingLayout;
    private TextView mStatusText;
    private Process mServerProcess;
    private int mServerPort = 0;
    private ValueCallback<Uri[]> mFilePathCallback;
    private final Handler mHandler = new Handler(Looper.getMainLooper());

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        setContentView(R.layout.activity_main);
        mWebView = findViewById(R.id.webview);
        mLoadingLayout = findViewById(R.id.loading_layout);
        mStatusText = findViewById(R.id.status_text);

        setupImmersiveMode();
        setupWebView();

        // Start Go backend in background thread
        new Thread(this::startGoBackend).start();
    }

    private void setupImmersiveMode() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            final WindowInsetsController insetsController = getWindow().getInsetsController();
            if (insetsController != null) {
                insetsController.setSystemBarsBehavior(WindowInsetsController.BEHAVIOR_DEFAULT);
            }
        }
    }

    private void setupWebView() {
        WebSettings settings = mWebView.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setDatabaseEnabled(true);
        settings.setAllowFileAccess(true);
        settings.setAllowContentAccess(true);
        settings.setUseWideViewPort(true);
        settings.setLoadWithOverviewMode(true);
        settings.setCacheMode(WebSettings.LOAD_DEFAULT);
        settings.setMediaPlaybackRequiresUserGesture(false);

        mWebView.setWebViewClient(new WebViewClient() {
            @Override
            public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
                String url = request.getUrl().toString();
                // Allow internal API and local web navigation inside the webview
                if (url.startsWith("http://127.0.0.1") || url.startsWith("http://localhost")) {
                    return false;
                }
                // Open external links (e.g. Lichess, Chess.com, TWIC) in system browser
                try {
                    Intent intent = new Intent(Intent.ACTION_VIEW, Uri.parse(url));
                    startActivity(intent);
                } catch (Exception e) {
                    Log.e(TAG, "Failed opening external link: " + url, e);
                }
                return true;
            }

            @Override
            public void onPageFinished(WebView view, String url) {
                super.onPageFinished(view, url);
                mHandler.post(() -> {
                    if (mLoadingLayout != null) {
                        mLoadingLayout.setVisibility(View.GONE);
                    }
                    mWebView.setVisibility(View.VISIBLE);
                });
            }
        });

        mWebView.setWebChromeClient(new WebChromeClient() {
            @Override
            public boolean onShowFileChooser(WebView webView, ValueCallback<Uri[]> filePathCallback, FileChooserParams fileChooserParams) {
                if (mFilePathCallback != null) {
                    mFilePathCallback.onReceiveValue(null);
                }
                mFilePathCallback = filePathCallback;

                Intent intent = fileChooserParams.createIntent();
                try {
                    startActivityForResult(intent, FILE_CHOOSER_REQUEST_CODE);
                } catch (Exception e) {
                    mFilePathCallback = null;
                    Toast.makeText(MainActivity.this, "Cannot open file picker", Toast.LENGTH_SHORT).show();
                    return false;
                }
                return true;
            }
        });
    }

    private int findFreePort() {
        try (ServerSocket socket = new ServerSocket(0)) {
            return socket.getLocalPort();
        } catch (IOException e) {
            return 8080;
        }
    }

    private void updateStatus(final String text) {
        mHandler.post(() -> {
            if (mStatusText != null) {
                mStatusText.setText(text);
            }
        });
    }

    private void startGoBackend() {
        try {
            updateStatus("Initializing NewScid engine...");
            mServerPort = findFreePort();

            File binDir = getFilesDir();
            File binFile = new File(binDir, "newscid-server");

            // Extract binary from assets if needed
            copyAssetToFile("newscid-server-arm64", binFile);
            if (!binFile.setExecutable(true, false)) {
                Log.w(TAG, "Failed to set binary executable permission");
            }

            updateStatus("Starting database and chess services...");
            ProcessBuilder pb = new ProcessBuilder(
                    binFile.getAbsolutePath(),
                    "serve",
                    "--host", "127.0.0.1",
                    "--port", String.valueOf(mServerPort),
                    "--data-dir", getFilesDir().getAbsolutePath()
            );

            pb.environment().put("NEWSCID_DATA_DIR", getFilesDir().getAbsolutePath());
            pb.environment().put("NEWSCID_PORT", String.valueOf(mServerPort));
            pb.directory(getFilesDir());

            mServerProcess = pb.start();
            Log.i(TAG, "Spawned Go server on port: " + mServerPort);

            // Wait for server healthcheck
            String targetUrl = "http://127.0.0.1:" + mServerPort + "/api/health";
            boolean ready = false;
            for (int i = 0; i < 40; i++) {
                try {
                    HttpURLConnection conn = (HttpURLConnection) new URL(targetUrl).openConnection();
                    conn.setConnectTimeout(300);
                    conn.setReadTimeout(300);
                    if (conn.getResponseCode() == 200) {
                        ready = true;
                        conn.disconnect();
                        break;
                    }
                    conn.disconnect();
                } catch (Exception ignored) {
                }
                Thread.sleep(150);
            }

            if (!ready) {
                updateStatus("Engine startup timed out. Retrying...");
            }

            mHandler.post(() -> {
                Log.i(TAG, "Loading NewScid Web UI at: http://127.0.0.1:" + mServerPort);
                mWebView.loadUrl("http://127.0.0.1:" + mServerPort);
            });

        } catch (Exception e) {
            Log.e(TAG, "Error starting backend: ", e);
            updateStatus("Startup failed: " + e.getMessage());
        }
    }

    private void copyAssetToFile(String assetName, File destFile) throws IOException {
        // If file already exists and is non-empty, keep it (or overwrite if app updated)
        if (destFile.exists() && destFile.length() > 0) {
            return;
        }

        try (InputStream in = getAssets().open(assetName);
             FileOutputStream out = new FileOutputStream(destFile)) {
            byte[] buffer = new byte[16384];
            int read;
            while ((read = in.read(buffer)) != -1) {
                out.write(buffer, 0, read);
            }
            out.flush();
        }
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        if (requestCode == FILE_CHOOSER_REQUEST_CODE) {
            if (mFilePathCallback == null) return;
            Uri[] results = null;
            if (resultCode == Activity.RESULT_OK && data != null) {
                String dataString = data.getDataString();
                if (dataString != null) {
                    results = new Uri[]{Uri.parse(dataString)};
                } else if (data.getClipData() != null) {
                    int count = data.getClipData().getItemCount();
                    results = new Uri[count];
                    for (int i = 0; i < count; i++) {
                        results[i] = data.getClipData().getItemAt(i).getUri();
                    }
                }
            }
            mFilePathCallback.onReceiveValue(results);
            mFilePathCallback = null;
        } else {
            super.onActivityResult(requestCode, resultCode, data);
        }
    }

    @Override
    public void onBackPressed() {
        if (mWebView != null && mWebView.canGoBack()) {
            mWebView.goBack();
        } else {
            super.onBackPressed();
        }
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        if (mServerProcess != null) {
            try {
                mServerProcess.destroy();
            } catch (Exception ignored) {
            }
        }
    }
}
