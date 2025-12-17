plugins {
    id 'com.android.application'
    id 'org.jetbrains.kotlin.android'
    id 'kotlin-kapt'
    id 'dagger.hilt.android.plugin'
    id 'kotlin-parcelize'
    id 'androidx.navigation.safeargs.kotlin'
}

android {
    namespace 'com.llmverifier.mobile.aurora'
    compileSdk 34

    defaultConfig {
        applicationId "com.llmverifier.mobile.aurora"
        minSdk 26  // Aurora OS minimum
        targetSdk 34
        versionCode 1
        versionName "1.0.0"

        testInstrumentationRunner "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        release {
            minifyEnabled true
            proguardFiles getDefaultProguardFile('proguard-android-optimize.txt'), 'proguard-rules.pro'
            buildConfigField "String", "BASE_URL", "\"http://localhost:8080\""
        }
        debug {
            buildConfigField "String", "BASE_URL", "\"http://10.0.2.2:8080\""
        }
    }

    compileOptions {
        sourceCompatibility JavaVersion.VERSION_1_8
        targetCompatibility JavaVersion.VERSION_1_8
    }

    kotlinOptions {
        jvmTarget = '1.8'
    }

    buildFeatures {
        compose true
        buildConfig true
    }

    composeOptions {
        kotlinCompilerExtensionVersion '1.5.8'
    }

    packagingOptions {
        resources {
            excludes += '/META-INF/{AL2.0,LGPL2.1}'
        }
    }
}

dependencies {
    // Core Android
    implementation 'androidx.core:core-ktx:1.12.0'
    implementation 'androidx.appcompat:appcompat:1.6.1'
    implementation 'com.google.android.material:material:1.11.0'
    implementation 'androidx.constraintlayout:constraintlayout:2.1.4'

    // Architecture Components
    implementation 'androidx.lifecycle:lifecycle-viewmodel-ktx:2.7.0'
    implementation 'androidx.lifecycle:lifecycle-livedata-ktx:2.7.0'
    implementation 'androidx.activity:activity-ktx:1.8.2'
    implementation 'androidx.fragment:fragment-ktx:1.6.2'

    // Dependency Injection
    implementation 'com.google.dagger:hilt-android:2.48'
    kapt 'com.google.dagger:hilt-compiler:2.48'

    // Networking
    implementation 'com.squareup.retrofit2:retrofit:2.9.0'
    implementation 'com.squareup.retrofit2:converter-gson:2.9.0'
    implementation 'com.squareup.okhttp3:logging-interceptor:4.12.0'

    // Reactive Programming
    implementation 'io.reactivex.rxjava3:rxjava:3.1.8'
    implementation 'io.reactivex.rxjava3:rxandroid:3.0.2'
    implementation 'com.squareup.retrofit2:adapter-rxjava3:2.9.0'

    // Database
    implementation 'androidx.room:room-runtime:2.6.1'
    implementation 'androidx.room:room-rxjava3:2.6.1'
    kapt 'androidx.room:room-compiler:2.6.1'

    // Security
    implementation 'androidx.security:security-crypto:1.1.0-alpha06'
    implementation 'androidx.biometric:biometric:1.1.0'

    // Work Manager
    implementation 'androidx.work:work-runtime-ktx:2.9.0'

    // Image Loading
    implementation 'com.github.bumptech.glide:glide:4.16.0'
    kapt 'com.github.bumptech.glide:compiler:4.16.0'

    // SwipeRefreshLayout
    implementation 'androidx.swiperefreshlayout:swiperefreshlayout:1.1.0'

    // Testing
    testImplementation 'junit:junit:4.13.2'
    androidTestImplementation 'androidx.test.ext:junit:1.1.5'
    androidTestImplementation 'androidx.test.espresso:espresso-core:3.5.1'
    androidTestImplementation 'androidx.test:runner:1.5.2'
}