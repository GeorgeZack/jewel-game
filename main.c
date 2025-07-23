/*
 * Simple Bejeweled Clone Scaffold with Hot Module Reloading (Raylib Version)
 * (Generated from ChatGPT on 07/19/25)
 */

#include <raylib.h>
#include <dlfcn.h>
#include <sys/stat.h>
#include <stdio.h>
#include <stdbool.h>
#include <time.h>
#include <unistd.h>
#include <stdlib.h>

#include "./game/game_logic.h"

typedef void (*gameInitFn)();
typedef void (*gameUpdateFn)();
typedef void (*gameRenderFn)();
typedef void (*gameCleanupFn)();

struct GameLib {
	void *handle;
	gameInitFn init;
	gameUpdateFn update;
	gameRenderFn render;
	gameCleanupFn cleanup;
	time_t last_mtime;
};

bool reloading = false;

bool loadGameLib(struct GameLib *lib, const char *path) {
	struct stat attr;
	if (stat(path, &attr) != 0) {
		printf("stat failed\n");
		return false;
	}

	void *handle = dlopen(path, RTLD_NOW);
	if (!handle) {
		fprintf(stderr, "dlopen failed: %s\n", dlerror());
		return false;
	}

	lib->handle = handle;
	lib->init = dlsym(handle, "GameLoopInit");
	lib->update = dlsym(handle, "GameLoopUpdate");
	lib->render = dlsym(handle, "GameLoopRender");
	lib->cleanup = dlsym(handle, "GameLoopCleanup");
	lib->last_mtime = attr.st_mtime;

	if (!lib->init || !lib->update || !lib->render || !lib->cleanup) {
		fprintf(stderr, "dlsym failed for one or more functions.\n");
		return false;
	}

	lib->init();
	return true;
}

void unloadGameLib(struct GameLib *lib) {
	if (lib->cleanup) lib->cleanup();
	if (lib->handle) dlclose(lib->handle);
	lib->handle = NULL;
}

bool checkForReload(struct GameLib *lib, const char *path) {
	struct stat attr;
	if (stat(path, &attr) != 0) return false;

	if (attr.st_mtime != lib->last_mtime) {
		unloadGameLib(lib);
		if (loadGameLib(lib, path)) {
			printf("Reloaded %s\n", path);
			reloading = true;
			return true;
		} else {
			printf("Failed to reload %s\n", path);
			reloading = false;
		}
	}
	return false;
}

int main() {
	InitWindow(640, 480, "Hot Swap Bejeweled Clone (Raylib)");
	SetTargetFPS(60);

	struct GameLib game = {0};
	const char *lib_path = "./game_logic.so";
	if (!loadGameLib(&game, lib_path)) {
		printf("Failed to load game logic.\n");
		return 1;
	}

	while (!WindowShouldClose()) {
		float delta = GetFrameTime();
		reloading = false;
		checkForReload(&game, lib_path);

		if (!reloading) {
			game.update();
			BeginDrawing();
			ClearBackground(DARKGRAY);
			game.render();
			EndDrawing();
		} else {
			BeginDrawing();
			ClearBackground(DARKGRAY);
			DrawText("RELOADING GAME LOGIC...", 2, 2, 10, WHITE);
			EndDrawing();
		}
	}

	unloadGameLib(&game);
	CloseWindow();

	return 0;
}

