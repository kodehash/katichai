package context

// Framework represents a detected framework
type Framework struct {
	Name     string
	Type     FrameworkType
	Version  string
	Language Language
}

// FrameworkType categorizes frameworks
type FrameworkType string

const (
	FrameworkTypeBackend   FrameworkType = "Backend"
	FrameworkTypeFrontend  FrameworkType = "Frontend"
	FrameworkTypeFullStack FrameworkType = "FullStack"
	FrameworkTypeUI        FrameworkType = "UI"
	FrameworkTypeTesting   FrameworkType = "Testing"
	FrameworkTypeORM       FrameworkType = "ORM"
	FrameworkTypeBuild     FrameworkType = "Build"
)

// Well-known frameworks
const (
	// Backend Frameworks
	FrameworkSpringBoot = "Spring Boot"
	FrameworkExpress    = "Express"
	FrameworkFastAPI    = "FastAPI"
	FrameworkGin        = "Gin"
	FrameworkDjango     = "Django"
	FrameworkFlask      = "Flask"
	FrameworkRails      = "Ruby on Rails"
	FrameworkLaravel    = "Laravel"
	FrameworkNestJS     = "NestJS"
	FrameworkKtor       = "Ktor"
	FrameworkActix      = "Actix"

	// Frontend/UI Frameworks
	FrameworkReact      = "React"
	FrameworkVue        = "Vue.js"
	FrameworkAngular    = "Angular"
	FrameworkSvelte     = "Svelte"
	FrameworkSolidJS    = "SolidJS"
	FrameworkPreact     = "Preact"
	FrameworkAlpineJS   = "Alpine.js"
	FrameworkLit        = "Lit"
	FrameworkEmber      = "Ember.js"
	FrameworkBackbone   = "Backbone.js"

	// Full-Stack/Meta Frameworks
	FrameworkNextJS     = "Next.js"
	FrameworkNuxt       = "Nuxt.js"
	FrameworkRemix      = "Remix"
	FrameworkSvelteKit  = "SvelteKit"
	FrameworkGatsby     = "Gatsby"
	FrameworkAstro      = "Astro"
	FrameworkQwik       = "Qwik"
	FrameworkSolidStart = "SolidStart"

	// UI Component Libraries
	FrameworkMaterialUI  = "Material-UI"
	FrameworkAntDesign   = "Ant Design"
	FrameworkChakraUI    = "Chakra UI"
	FrameworkTailwindCSS = "Tailwind CSS"
	FrameworkBootstrap   = "Bootstrap"
	FrameworkBulma       = "Bulma"
	FrameworkFoundation  = "Foundation"
	FrameworkShadcnUI    = "shadcn/ui"
	FrameworkDaisyUI     = "DaisyUI"
	FrameworkMantine     = "Mantine"
	FrameworkPrimeReact  = "PrimeReact"
	FrameworkVuetify     = "Vuetify"
	FrameworkQuasar      = "Quasar"

	// Mobile Frameworks
	FrameworkReactNative = "React Native"
	FrameworkFlutter     = "Flutter"
	FrameworkIonic       = "Ionic"
	FrameworkCapacitor   = "Capacitor"

	// Testing Frameworks
	FrameworkJest       = "Jest"
	FrameworkMocha      = "Mocha"
	FrameworkJUnit      = "JUnit"
	FrameworkPytest     = "Pytest"
	FrameworkVitest     = "Vitest"
	FrameworkPlaywright = "Playwright"
	FrameworkCypress    = "Cypress"

	// Build Tools
	FrameworkWebpack = "Webpack"
	FrameworkVite    = "Vite"
	FrameworkRollup  = "Rollup"
	FrameworkParcel  = "Parcel"
	FrameworkEsbuild = "esbuild"
	FrameworkTurbo   = "Turbopack"
)

// FrameworkInfo contains metadata about a framework
type FrameworkInfo struct {
	Name        string
	Type        FrameworkType
	Language    Language
	Indicators  []string // File/directory patterns
	PackageKeys []string // Package.json/requirements.txt keys
}

// GetFrameworkRegistry returns all known frameworks
func GetFrameworkRegistry() []FrameworkInfo {
	return []FrameworkInfo{
		// Backend Frameworks
		{
			Name:        FrameworkSpringBoot,
			Type:        FrameworkTypeBackend,
			Language:    LanguageJava,
			Indicators:  []string{"@SpringBootApplication", "@RestController", "@Service"},
			PackageKeys: []string{"spring-boot-starter"},
		},
		{
			Name:        FrameworkExpress,
			Type:        FrameworkTypeBackend,
			Language:    LanguageJavaScript,
			Indicators:  []string{"express()", "app.use("},
			PackageKeys: []string{"express"},
		},
		{
			Name:        FrameworkFastAPI,
			Type:        FrameworkTypeBackend,
			Language:    LanguagePython,
			Indicators:  []string{"from fastapi import", "FastAPI("},
			PackageKeys: []string{"fastapi"},
		},
		{
			Name:        FrameworkGin,
			Type:        FrameworkTypeBackend,
			Language:    LanguageGo,
			Indicators:  []string{"gin.Default()", "gin.New()", ".GET(", ".POST("},
			PackageKeys: []string{"github.com/gin-gonic/gin"},
		},
		{
			Name:        FrameworkDjango,
			Type:        FrameworkTypeBackend,
			Language:    LanguagePython,
			Indicators:  []string{"from django", "INSTALLED_APPS", "settings.py"},
			PackageKeys: []string{"django"},
		},
		{
			Name:        FrameworkFlask,
			Type:        FrameworkTypeBackend,
			Language:    LanguagePython,
			Indicators:  []string{"from flask import", "Flask(__name__)"},
			PackageKeys: []string{"flask"},
		},
		{
			Name:        FrameworkNestJS,
			Type:        FrameworkTypeBackend,
			Language:    LanguageTypeScript,
			Indicators:  []string{"@Module(", "@Controller(", "@Injectable("},
			PackageKeys: []string{"@nestjs/core"},
		},

		// Frontend/UI Frameworks
		{
			Name:        FrameworkReact,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"import React", "from 'react'", "useState", "useEffect"},
			PackageKeys: []string{"react"},
		},
		{
			Name:        FrameworkVue,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"<template>", "export default {", "Vue.component"},
			PackageKeys: []string{"vue"},
		},
		{
			Name:        FrameworkAngular,
			Type:        FrameworkTypeUI,
			Language:    LanguageTypeScript,
			Indicators:  []string{"@Component(", "@NgModule(", "angular.json"},
			PackageKeys: []string{"@angular/core"},
		},
		{
			Name:        FrameworkSvelte,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"<script>", "<style>", ".svelte"},
			PackageKeys: []string{"svelte"},
		},
		{
			Name:        FrameworkSolidJS,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"import { createSignal", "from 'solid-js'"},
			PackageKeys: []string{"solid-js"},
		},
		{
			Name:        FrameworkPreact,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"from 'preact'", "import { h }"},
			PackageKeys: []string{"preact"},
		},
		{
			Name:        FrameworkAlpineJS,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"x-data", "x-bind", "x-on"},
			PackageKeys: []string{"alpinejs"},
		},

		// Full-Stack/Meta Frameworks
		{
			Name:        FrameworkNextJS,
			Type:        FrameworkTypeFullStack,
			Language:    LanguageJavaScript,
			Indicators:  []string{"next.config", "pages/", "app/"},
			PackageKeys: []string{"next"},
		},
		{
			Name:        FrameworkNuxt,
			Type:        FrameworkTypeFullStack,
			Language:    LanguageJavaScript,
			Indicators:  []string{"nuxt.config", "pages/", ".nuxt/"},
			PackageKeys: []string{"nuxt"},
		},
		{
			Name:        FrameworkRemix,
			Type:        FrameworkTypeFullStack,
			Language:    LanguageTypeScript,
			Indicators:  []string{"remix.config", "app/routes/"},
			PackageKeys: []string{"@remix-run/react"},
		},
		{
			Name:        FrameworkSvelteKit,
			Type:        FrameworkTypeFullStack,
			Language:    LanguageJavaScript,
			Indicators:  []string{"svelte.config", "src/routes/"},
			PackageKeys: []string{"@sveltejs/kit"},
		},
		{
			Name:        FrameworkGatsby,
			Type:        FrameworkTypeFullStack,
			Language:    LanguageJavaScript,
			Indicators:  []string{"gatsby-config", "gatsby-node"},
			PackageKeys: []string{"gatsby"},
		},
		{
			Name:        FrameworkAstro,
			Type:        FrameworkTypeFullStack,
			Language:    LanguageJavaScript,
			Indicators:  []string{"astro.config", ".astro"},
			PackageKeys: []string{"astro"},
		},

		// UI Component Libraries
		{
			Name:        FrameworkMaterialUI,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"@mui/material", "from '@mui/"},
			PackageKeys: []string{"@mui/material"},
		},
		{
			Name:        FrameworkAntDesign,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"from 'antd'", "import { Button } from 'antd'"},
			PackageKeys: []string{"antd"},
		},
		{
			Name:        FrameworkChakraUI,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"@chakra-ui", "from '@chakra-ui/react'"},
			PackageKeys: []string{"@chakra-ui/react"},
		},
		{
			Name:        FrameworkTailwindCSS,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"tailwind.config", "@tailwind", "className="},
			PackageKeys: []string{"tailwindcss"},
		},
		{
			Name:        FrameworkShadcnUI,
			Type:        FrameworkTypeUI,
			Language:    LanguageTypeScript,
			Indicators:  []string{"components/ui/", "cn(", "class-variance-authority"},
			PackageKeys: []string{"class-variance-authority"},
		},
		{
			Name:        FrameworkMantine,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"@mantine/core", "from '@mantine/"},
			PackageKeys: []string{"@mantine/core"},
		},
		{
			Name:        FrameworkVuetify,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"vuetify", "<v-app>", "<v-btn>"},
			PackageKeys: []string{"vuetify"},
		},

		// Mobile Frameworks
		{
			Name:        FrameworkReactNative,
			Type:        FrameworkTypeUI,
			Language:    LanguageJavaScript,
			Indicators:  []string{"react-native", "from 'react-native'", "<View>"},
			PackageKeys: []string{"react-native"},
		},

		// Build Tools
		{
			Name:        FrameworkVite,
			Type:        FrameworkTypeBuild,
			Language:    LanguageJavaScript,
			Indicators:  []string{"vite.config", "import.meta.env"},
			PackageKeys: []string{"vite"},
		},
		{
			Name:        FrameworkWebpack,
			Type:        FrameworkTypeBuild,
			Language:    LanguageJavaScript,
			Indicators:  []string{"webpack.config", "module.exports"},
			PackageKeys: []string{"webpack"},
		},
	}
}
