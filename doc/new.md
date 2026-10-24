# pm new : l'assistant

```powershell
.\pm.exe new [config.yaml] [--lang fr|en]
```

`pm new` (ou un double-clic sur `pm.exe`, puis « créer une config ») pose quelques questions, une à la fois, et écrit une config qui marche, avec ses raccourcis.
La config n'est écrite qu'après la confirmation finale, qui résume tout en clair.
Pour démarrer avec pm, c'est le plus simple : pas de YAML à écrire.

## Ce qu'il sait faire {#cases}

| Choix | Ce qu'on obtient |
|---|---|
| lancer DCS World avec des profils | une config `pm run` par profils, réglée pour DCS : lanceur, `dcs.exe` suivi, SRS arrêté à la fermeture si on le veut |
| lancer un autre jeu avec des profils | la même chose pour n'importe quel jeu : on donne le lanceur, le process du jeu s'il prend le relais, et les dossiers où le jeu range ses réglages |
| garder un programme en vie | une config `pm watch` : instance déjà lancée arrêtée, fichiers à supprimer avant chaque lancement, délai de relance |
| sauvegarder avant de lancer | une archive datée d'un dossier avant chaque lancement, en gardant les plus récentes |

Les questions communes viennent ensuite : le nom de la config (`dcs`, qui donne aussi leur nom aux raccourcis : « dcs - DEV »), le fichier de config à créer (proposé, modifiable) et l'emplacement des [raccourcis](shortcut.md) : à côté de la config, sur le Bureau, dans le menu Démarrer ou dans un autre dossier.

`pm new` ne modifie jamais une config : si le fichier existe déjà, il demande un autre nom.

## Les profils se règlent en jouant {#learn}

Pour un jeu à profils, l'assistant ne demande pas quels fichiers emporter : il le découvre.

1. **Les profils** : on tape leurs noms, un par ligne (`DEV`, `VR`, `STD`).
   Chaque profil a son dossier à côté de la config, du même nom ; l'assistant refuse un nom dont le dossier existe déjà, pour ne jamais écrire sur des fichiers qu'il n'a pas créés.
2. **Les réglages actuels sont mis de côté** : une copie des dossiers de réglages du jeu (pour DCS : `Config` et `Scripts` dans Saved Games), dans un dossier `.pm-new` à côté de la config.
   Pour DCS, `Config\authdata.bin` (les données de connexion, que DCS réécrit à chaque lancement) est laissé de côté : il n'est ni copié, ni remis, ni emporté.
3. **Le profil des réglages actuels** : une seule question, « quel profil correspond aux réglages actuels ? ». Ce profil les prend tels quels, sans lancer le jeu ; « aucun » fait tout régler en jouant.
4. **Les autres profils, l'un après l'autre** : les réglages d'origine sont remis, puis l'assistant demande s'il faut partir de fichiers existants :
    - d'une **ancienne config pm** (un profil de `pm-dcs.yaml`, par exemple) : ses lignes `sync` disent où va chaque fichier, et les fichiers du profil sont cherchés d'abord à côté du `.yaml` (une sauvegarde déplacée garde les chemins de son ancien emplacement) ;
    - d'un **dossier rangé comme le jeu** (pour DCS : `Config\...`, `Scripts\...`) : chaque fichier va au même chemin ;
    - de **fichiers choisis un par un** : pour chacun, on choisit où il va dans la liste des fichiers du même nom déjà dans le jeu, le plus probable en premier (le `Throttle.diff.lua` d'un dossier `F-16C_50` va vers celui du F-16 ; à défaut, le chemin le moins profond, `Config\options.lua` avant les presets), ou on tape un autre chemin.

   Les fichiers hors des dossiers surveillés sont laissés de côté, et la console le dit.
   Ensuite le jeu démarre, et l'assistant affiche : *tout régler comme voulu pour ce profil, puis quitter le jeu*.
   À la fermeture, il relève les fichiers modifiés, créés ou supprimés, en comparant leur contenu, et les présente aussitôt, avec des noms parlants pour les fichiers de DCS (« options du jeu », « commandes de … »).
   On décoche ce qui ne devait pas changer : ce profil en garde la version d'origine.
5. **Ce qui est emporté** : chaque fichier qu'au moins un profil a changé.
   Chaque profil garde sa version ; un profil qui n'y a pas touché garde celle d'origine.
   Un fichier qu'un profil n'a pas (une manette ajoutée dans un seul profil) est supprimé avant le lancement de ce profil, pour qu'il ne prenne pas celui d'un autre.
6. **La confirmation** montre le résultat en clair : les fichiers pareils dans tous les profils, puis ceux propres à chaque profil.
7. **À la fin, les réglages d'origine sont remis** : après l'assistant, le jeu est réglé comme avant.
8. **Une icône par profil** : l'assistant propose les `.ico` trouvés à côté de la config et dans ses sous-dossiers (jusqu'à trois niveaux), celle qui porte le nom du profil en premier (`dcs-dev.ico` pour DEV) ; ou aucune, ou un autre fichier. Elle s'affiche sur les raccourcis et dans la zone de notification.

À la confirmation, l'assistant écrit la config, remplit le dossier de chaque profil avec ses fichiers, crée les raccourcis et supprime `.pm-new`.

## Les touches {#keys}

| Touche | Effet |
|---|---|
| Entrée | valider ; sur une question vide, la valeur proposée |
| ↑ ↓ | choisir dans une liste |
| Espace | cocher ou décocher un fichier |
| Échap | revenir à la question précédente |
| Ctrl+C | abandonner |

On ne revient pas en arrière au-delà d'une étape qui a touché aux fichiers (la mise de côté des réglages, une partie jouée pour un profil).

## Si l'assistant s'arrête en route {#interrupted}

Abandonner (Ctrl+C sur une question) remet les réglages d'origine et n'écrit rien.
Pendant qu'un jeu tourne pour un profil, Ctrl+C n'a pas d'effet : arrêter l'assistant laisserait le jeu ouvert, et il réécrirait ses fichiers en se fermant ; il faut quitter le jeu.
Si l'assistant n'a pas pu les remettre (fenêtre fermée, panne), leur copie reste dans `.pm-new` : relancer `pm new` avec le même fichier de config les remet avant toute autre chose.
