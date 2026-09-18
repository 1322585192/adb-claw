package observe

import "testing"

func TestDiffTreesSame(t *testing.T) {
	tree, err := ParseUITree([]byte(sampleXML))
	if err != nil {
		t.Fatal(err)
	}
	delta := DiffTrees(tree, tree)
	if !delta.Same {
		t.Fatalf("expected same, got %+v", delta)
	}
}

func TestDiffTreesAddedRemoved(t *testing.T) {
	prev, err := ParseUITree([]byte(sampleXML))
	if err != nil {
		t.Fatal(err)
	}
	currXML := `<?xml version="1.0" encoding="UTF-8"?>
<hierarchy rotation="0">
  <node index="0" text="New" resource-id="" class="android.widget.Button" package="com.example" content-desc="" checkable="false" checked="false" clickable="true" enabled="true" focusable="true" focused="false" scrollable="false" selected="false" bounds="[0,0][100,100]"></node>
</hierarchy>`
	curr, err := ParseUITree([]byte(currXML))
	if err != nil {
		t.Fatal(err)
	}
	delta := DiffTrees(prev, curr)
	if delta.Same {
		t.Fatal("expected a delta")
	}
	if len(delta.Added) == 0 {
		t.Fatal("expected added nodes")
	}
	if len(delta.Removed) == 0 {
		t.Fatal("expected removed nodes")
	}
}
