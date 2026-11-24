# Exemple : DCS World par profils

**Le besoin** : jouer à DCS World avec plusieurs jeux de réglages (un profil pour l'écran, un pour la réalité virtuelle, un pour le développement de missions), sans les recopier à la main, et sans qu'un profil écrase l'autre.

**Ce que fait Process Monitor** :

1. il copie les fichiers du profil choisi (`options.lua`, `export.lua`, des assignations de commandes) à leur place dans `Saved Games\DCS` ;
2. il lance `dcs_updater.exe`, puis suit `dcs.exe`, que le lanceur démarre avant de se fermer ;
3. quand DCS est fermé, il ferme aussi SRS s'il tourne, puis recopie les fichiers dans le profil, avec les réglages changés en jeu ;
4. si DCS tourne déjà, ou si un autre profil est en cours, il refuse de démarrer.

## Préparer les profils {#profiles}

Un dossier par profil dans `Saved Games\DCS\Profiles`, avec une copie des fichiers à faire suivre :

```text
Saved Games\DCS\Profiles\
  DEV\  export.lua  options.lua  joystick.diff.lua
  VR\   export.lua  options.lua
  STD\  export.lua  options.lua
  Icons\ dcs-dev.ico  dcs-vr.ico  dcs.ico
```

Pour créer un profil, copier les fichiers actuels de DCS (`Scripts\export.lua`, `Config\options.lua`…) dans son dossier.

## La config {#config}

```yaml
--8<-- "examples/dcs.yaml"
```

Ligne par ligne :

- `name: dcs` : les trois profils partagent un même verrou ; impossible d'en lancer deux à la fois.
- `prof` vaut le dossier du profil choisi avec `--profile`.
- `track: dcs.exe` : la session suit le jeu, pas le lanceur.
- `kill_on_exit` : SRS, lancé par DCS, est fermé avec lui.
- `ready` : le journal note quand DCS a fini de démarrer (le fichier `dcs.lock` apparaît).
- `sync` : deux fichiers communs à tous les profils ; DEV en ajoute un troisième.
- `ui: tray` et `icon:` par profil : pas de fenêtre console, une icône à l'image du profil.
- `lang: fr` : pm parle français, même sur un Windows en anglais.

## Vérifier, puis lancer {#run}

```powershell
.\pm.exe check .\dcs.yaml --profile DEV
.\pm.exe run .\dcs.yaml --profile DEV
```

`pm check` signale un fichier de profil manquant avant qu'on lance quoi que ce soit.

## Un raccourci par profil {#shortcuts}

La section `shortcuts` de la config les décrit ; une commande les crée tous :

```powershell
.\pm.exe shortcut .\dcs.yaml
```

Chacun a ces réglages :

- **Cible** : `C:\...\pm.exe run "C:\...\dcs.yaml" --profile DEV`
- **Démarrer dans** : le dossier de la config
- **Exécuter** : Réduite
- **Icône** : la même que celle du profil (`Icons\dcs-dev.ico`)

Un double-clic lance DCS avec ce profil ; l'icône apparaît dans la zone de notification et disparaît quand les réglages sont récupérés.

## DCS et SteamVR sur des cœurs séparés {#steamvr}

**Le besoin** : en réalité virtuelle, laisser à DCS ses cœurs, et faire tourner SteamVR sur les autres.

**Pourquoi par Process Monitor** : la clé `affinity_mask` d'`autoexec.cfg` n'a aucun effet dans DCS 2.9.31. Le fichier `dcs.log` indique toujours tous les cœurs, quelle que soit sa valeur. DCS lit, à son démarrage, les cœurs que Windows a donnés à son process. Il faut donc les poser avant son lancement, et non après : c'est ce que fait le bloc `affinity`, qui remplace `affinity_mask`.

Pour un processeur à 16 processeurs logiques (8 cœurs avec SMT), dans le profil VR de la config ci-dessus :

```yaml
profiles:
  VR:
    icon: ${saved}\Profiles\Icons\dcs-vr.ico
    affinity:
      cores: 0-11
      others:
        cores: 12-15
        processes: [vrserver.exe, vrcompositor.exe, vrmonitor.exe, vrdashboard.exe, vrwebhelper.exe, vrstartup.exe]
```

- DCS, et ce qu'il démarre, tourne sur les cœurs 0 à 11 ; les profils STD et DEV gardent tous les cœurs.
- Les process de SteamVR sont tenus sur les cœurs 12 à 15 pendant la session, puis retrouvent leurs cœurs. Si SteamVR tourne déjà, il est déplacé avant que DCS démarre.
- La liste vient de SteamVR 2.19 avec Steam Link. Un autre casque ou un autre runtime peut ajouter ses propres process : les chercher dans le Gestionnaire des tâches, onglet Détails, pendant une session VR.
- Garder des cœurs physiques entiers : `12-15`, pas `13-15`.

**Vérifier** : après un lancement par Process Monitor, la ligne `process affinity mask:` en tête de `Saved Games\DCS\Logs\dcs.log` compte un chiffre par cœur. Avec `0-11`, elle vaut `111111111111` (12 chiffres). Seize `1` veulent dire que DCS a reçu tous les cœurs : il n'a pas été lancé par Process Monitor, ou un autre programme l'a démarré (le journal de Process Monitor le signale alors).

```powershell
Select-String -Path "$env:USERPROFILE\Saved Games\DCS\Logs\dcs.log" -Pattern "process affinity mask" | Select-Object -First 1
```

Pour SteamVR, le journal de Process Monitor note chaque process placé (`affinité : vrserver.exe (pid …) placé sur les cœurs 12-15`).

**Mesurer le gain (facultatif)** : rien ne garantit qu'il existe. Sur un processeur à une seule puce, dont le cache est commun à tous les cœurs, il sera modeste, et DCS perd 4 processeurs logiques. Pour en juger :

1. Rejouer deux fois la même piste (`.trk`) en VR, une fois avec le bloc `affinity`, une fois sans.
2. Relever les temps d'image pendant trois minutes à chaque passage, avec CapFrameX ou fpsVR.
3. Comparer la moyenne et le 1 % le plus lent.
