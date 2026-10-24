# pm shortcut : les raccourcis

```powershell
.\pm.exe shortcut <config.yaml> [--profile NOM]
```

`pm shortcut` crée les raccourcis Windows d'une config : un par profil, ou un seul sans profils, chacun lançant `pm.exe` sur cette config.
Avec `--profile`, seulement celui de ce profil.

## La section shortcuts {#section}

La config dit quels raccourcis elle veut :

```yaml
shortcuts:
  mode: run                   # run ou watch : la commande que le raccourci lance (obligatoire)
  name: DCS - ${profile}      # nom du raccourci, sans .lnk
  folder: ${start_menu}       # dossier où le poser
```

- **`name`** : par défaut `${name}`, ou `${name} - ${profile}` quand la config a des profils.
- **`folder`** : un dossier, ou une liste pour poser chaque raccourci à plusieurs endroits (`[${desktop}, ${start_menu}]`) ; par défaut le dossier de la config (`${config_dir}`). Deux variables donnent les dossiers habituels : `${desktop}` (le Bureau) et `${start_menu}` (le menu Démarrer). Windows les connaît même quand le Bureau a été déplacé dans OneDrive.
- Sans section `shortcuts`, `pm shortcut` le dit et ne crée rien. Depuis le [menu](installation.md#menu) (double-clic sur `pm.exe`), il propose de l'ajouter : deux questions (le mode, puis un ou plusieurs dossiers, à cocher), puis la section est écrite à la fin du fichier, sans toucher au reste.
  Si la config ne dit pas comment `pm` s'affiche (pas de clé `ui`), le menu le demande — icône de notification ou fenêtre console — et ajoute la ligne : c'est ce qui fait ouvrir le raccourci « Réduite ».
  Le menu propose aussi une icône à chaque profil qui n'en a pas, parmi les `.ico` trouvés à côté de la config : `DEV: {}` devient `DEV:` suivi d'une ligne `icon:`, et rien d'autre ne change.

## Le raccourci créé {#link}

| Réglage | Valeur |
|---|---|
| Cible | le `pm.exe` qui a lancé la commande, avec `run` ou `watch`, la config et `--profile` |
| Démarrer dans | le dossier de la config |
| Exécuter | « Réduite » quand la config utilise l'[icône de notification](tray.md) (`ui: tray`), « Fenêtre normale » sinon |
| Icône | l'icône du profil ou de la config si c'est un `.ico`, sinon celle du programme lancé |

Un raccourci du même nom déjà présent est gardé en `<nom>.lnk.old` avant d'être remplacé : pour revenir en arrière, renommer le `.old`.

**La barre des tâches** : Windows ne laisse aucun programme y épingler un raccourci sans demander. Quand des raccourcis sont posés dans le menu Démarrer, `pm shortcut` le rappelle : clic droit sur le raccourci dans le menu Démarrer, puis « Épingler à la barre des tâches ».

**Déplacer `pm.exe`** casse ses raccourcis : relancer `pm shortcut` depuis son nouvel emplacement les refait.
