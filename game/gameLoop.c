#include "gameLoop.h"
#include "ui.c"
#include <raylib.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

Gem createNewGem(bool atMouse) {
	Gem newGem;
	newGem.texture = textures[GetRandomValue(0, MAX_TEXTURES)];
	newGem.color = (Color){ GetRandomValue(50, 250), GetRandomValue(50, 250), GetRandomValue(50, 250), 250 };
	newGem.discoloration.r = GetRandomValue(-250, 250) / 60.0f;
	newGem.discoloration.g = GetRandomValue(-250, 250) / 60.0f;
	newGem.discoloration.b = GetRandomValue(-250, 250) / 60.0f;
	newGem.discoloration.a = GetRandomValue(-50, 50) / 60.0f;
	if (atMouse) {
		int x, y;
		int bx = newGem.texture.width / 2.0f + 5;
		int by = newGem.texture.height / 2.0f + 5;

		if (GetMouseX() > 0 + bx && GetMouseX() < GetScreenWidth() - bx) {
			x = GetMouseX() - newGem.texture.width / 2.0f;
		} else if (GetMouseX() <= 0 + bx){
			x = bx / 2;
		} else if (GetMouseX() >= GetScreenWidth() - bx) {
			x = GetScreenWidth() - (bx * 2);
		} else {
			x = 0;
		}

		if (GetMouseY() > 0 + by && GetMouseY() < GetScreenHeight() - by) {
			y = GetMouseY() - newGem.texture.height / 2.0f;
		} else if (GetMouseY() <= 0 + by){
			y = by / 2;
		} else if (GetMouseY() >= GetScreenHeight() - by) {
			y = GetScreenHeight() - (by * 2);
		} else {
			y = 0;
		}

		newGem.position = (Vector2){ x, y };
	} else {
		newGem.position = (Vector2){ (GetScreenWidth() / 2.0f) - (newGem.texture.width / 2.0f), (GetScreenHeight() / 2.0f) - (newGem.texture.height / 2.0f) };
	}
	newGem.speed.x = (float)GetRandomValue(-250, 250) / 60.0f;
	newGem.speed.y = (float)GetRandomValue(-250, 250) / 60.0f;
	gemCount++;
	return newGem;
}

void discolorGem(Gem *gem) {
	gem->color.r += gem->discoloration.r;
	gem->color.g += gem->discoloration.g;
	gem->color.b += gem->discoloration.b;
	gem->color.a += gem->discoloration.a;
	if ((gem->color.r <= 50 ) || (gem->color.r >= 250)) gem->discoloration.r *= -1;
	if ((gem->color.g <= 50 ) || (gem->color.g >= 250)) gem->discoloration.g *= -1;
	if ((gem->color.b <= 50 ) || (gem->color.b >= 250)) gem->discoloration.b *= -1;
	if ((gem->color.b <= 50 ) || (gem->color.b >= 250)) gem->discoloration.a *= -1;
}

void moveGem(Gem *gem) {
	gem->position.x += gem->speed.x;
	gem->position.y += gem->speed.y;

	if (((gem->position.x + gem->texture.width - 5) > GetScreenWidth()) ||
		((gem->position.x + 5) < 0)) gem->speed.x *= -1;
	if (((gem->position.y + gem->texture.height - 5) > GetScreenHeight()) ||
		((gem->position.y + 5) < 0)) gem->speed.y *= -1;
}

void renderGem(Gem *gem) {
	if (gem->texture.id != 0) {
		DrawTexture(gem->texture, (int)gem->position.x, (int)gem->position.y, gem->color);
	}
}

////////////////////////////////////////////////////////////////////////////////
// Game Loop Methods
////////////////////////////////////////////////////////////////////////////////

void GameLoopInit() {
	textures[0] = LoadTexture("_res/amethyst.png");
	textures[1] = LoadTexture("_res/crystalize.png");
	textures[2] = LoadTexture("_res/cut-diamond.png");
	textures[3] = LoadTexture("_res/emerald.png");
	textures[4] = LoadTexture("_res/rupee.png");
	textures[5] = LoadTexture("_res/saphir.png");
	textures[6] = LoadTexture("_res/topaz.png");

	gems = (Gem *)RL_MALLOC(sizeof(Gem) * MAX_GEMS);
	if (gems == NULL) {
		fprintf(stderr, "GameLoopInit: Failed to allocate memory for gems!\n");
	} else {
		memset(gems, 0, sizeof(Gem) * MAX_GEMS);
	}
	gemCount = 0;

	InitUI();
}

void GameLoopUpdate() {
	if (gems == NULL) return;

	if (gemCount == 0) {
		gems[0] = createNewGem(false);
	}

	if (IsMouseButtonDown(MOUSE_BUTTON_LEFT)) {
		for (int i = 0; i < 100 && gemCount < MAX_GEMS; i++) {
			gems[gemCount] = createNewGem(true);
		}
	}

	for (int i = 0; i < gemCount; i++) {
		discolorGem(&gems[i]);
		moveGem(&gems[i]);
	}
}

void GameLoopRender() {
	if (gems != NULL) {
		for (int i = 0; i < gemCount; i++) {
			renderGem(&gems[i]);
		}
	}
	DrawUI();
}

void GameLoopCleanup() {
	for (int i = 0; i < MAX_TEXTURES; i++) {
		Texture2D tex = textures[i];
		if (tex.id != 0) {
			UnloadTexture(tex);
			tex.id = 0;
		}
	}

	if (gems != NULL) {
		RL_FREE(gems);
		gems = NULL;
	}

	gemCount = 0;
}

