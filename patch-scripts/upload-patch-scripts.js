import { createClient } from '@supabase/supabase-js';
import fs from 'fs';
import path from 'path';

const supabaseUrl = process.env.SUPABASE_URL || 'http://localhost:54321';
const supabaseServiceKey = process.env.SUPABASE_SERVICE_ROLE_KEY;

if (!supabaseServiceKey) {
  throw new Error('SUPABASE_SERVICE_ROLE_KEY is required');
}

const supabase = createClient(supabaseUrl, supabaseServiceKey);

const SCRIPTS_DIR = './patch-scripts';
const PATCH_SCRIPTS_BUCKET = 'patch-scripts';
const PATCH_OUTPUTS_BUCKET = 'patch-execution-outputs';

async function uploadPatchScripts() {
  try {
    console.log('Starting patch scripts upload...\n');
    
    // Ensure patch-scripts directory exists
    if (!fs.existsSync(SCRIPTS_DIR)) {
      console.log(`Creating ${SCRIPTS_DIR} directory...`);
      fs.mkdirSync(SCRIPTS_DIR, { recursive: true });
      console.log(`\n✓ ${SCRIPTS_DIR} directory created. Please add your patch scripts here.`);
      return;
    }

    // Recursively get all files in patch-scripts directory
    const files = getAllFiles(SCRIPTS_DIR);

    if (files.length === 0) {
      console.log(`No patch scripts found in ${SCRIPTS_DIR}`);
      return;
    }

    let uploadedCount = 0;
    let failedCount = 0;

    for (const filePath of files) {
      const fileContent = fs.readFileSync(filePath);
      const relativePath = path.relative(SCRIPTS_DIR, filePath);
      
      console.log(`Uploading ${relativePath} to ${PATCH_SCRIPTS_BUCKET}...`);
      
      const { data, error } = await supabase.storage
        .from(PATCH_SCRIPTS_BUCKET)
        .upload(relativePath, fileContent, {
          upsert: true,
          contentType: 'application/x-sh',
        });

      if (error) {
        console.error(`✗ Failed to upload ${relativePath}:`, error.message);
        failedCount++;
      } else {
        console.log(`✓ Uploaded ${relativePath}`);
        uploadedCount++;
      }
    }

    console.log(`\n========================================`);
    console.log(`Upload Summary:`);
    console.log(`  Successful: ${uploadedCount}`);
    console.log(`  Failed: ${failedCount}`);
    console.log(`  Bucket: ${PATCH_SCRIPTS_BUCKET}`);
    console.log(`========================================\n`);

    if (failedCount > 0) {
      process.exit(1);
    }
  } catch (error) {
    console.error('Error uploading patch scripts:', error);
    process.exit(1);
  }
}

function getAllFiles(dir) {
  let files = [];
  const items = fs.readdirSync(dir);

  for (const item of items) {
    const fullPath = path.join(dir, item);
    const stat = fs.statSync(fullPath);

    if (stat.isDirectory()) {
      files = files.concat(getAllFiles(fullPath));
    } else {
      files.push(fullPath);
    }
  }

  return files;
}

uploadPatchScripts();
