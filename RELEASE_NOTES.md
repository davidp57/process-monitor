# pm 2.0.0

**process-monitor devient pm : un seul fichier `pm.exe`, plus fiable, qui ne risque plus d'écraser les réglages d'un profil.**

---

## ⚠️ Avant d'installer

Cette version remplace l'ancien process-monitor, et **ses fichiers de configuration ne sont plus lus tels quels**.
Il faut réécrire chaque config au nouveau format : la page [Migration depuis la 1.x](https://davidp57.github.io/process-monitor/latest/migration/) donne la correspondance option par option.
Les raccourcis qui lançaient `process-monitor.exe --config …` doivent ensuite pointer sur `pm.exe run …` ou `pm.exe watch …`.

## Deux usages, deux commandes

- **`pm run`** lance un programme une fois, avec des actions avant et après : typiquement, mettre en place les réglages d'un profil, lancer le jeu, puis récupérer les réglages à sa fermeture.
- **`pm watch`** garde un programme en vie et le relance s'il s'arrête.
- **`pm check`** montre ce que ferait une config, sans rien lancer : à passer après chaque modification.

## Vos profils sont protégés

- Les fichiers d'un profil s'écrivent sur **une seule ligne** : copiés au lancement, récupérés à la fermeture.
- pm **refuse de démarrer** si le jeu tourne déjà, ou si un autre profil du même jeu est en cours : plus de réglages d'un profil recopiés dans un autre.
- La récupération des réglages se fait **une seule fois**, et seulement quand le jeu est vraiment fermé, même s'il est passé par un lanceur.
- Si le démarrage échoue à mi-chemin, les fichiers déjà copiés sont remis comme avant.

## Sans fenêtre console

Avec `--tray`, pm ne laisse plus de fenêtre ouverte : une **icône dans la zone de notification** montre l'état (vert, orange, rouge) et son menu permet d'ouvrir le journal ou d'arrêter.
L'icône peut prendre l'image de votre choix, par exemple celle de votre raccourci, avec une pastille de couleur pour l'état.
Plus de fenêtre à fermer par mégarde, ce qui faisait perdre la récupération des réglages.

## Et aussi

- **En français et en anglais**, au choix (`--lang fr`, ou `lang: fr` dans la config).
- Les programmes que le jeu ouvre à côté (un client radio comme SRS) ne retardent plus la récupération des réglages, et peuvent être fermés automatiquement avec le jeu.
- `q` ne coupe plus le programme : il détache pm, après confirmation ; `k` arrête.

---

📖 [Documentation](https://davidp57.github.io/process-monitor/latest/) · 🇬🇧 [English](https://davidp57.github.io/process-monitor/latest/en/)
